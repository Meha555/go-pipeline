// 通过操作YAML的AST来实现include文件的合并
package parser

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const includesKey = "includes"

func loadConfigNode(configPath string) (*yaml.Node, error) {
	node, _, err := loadConfigNodeWithStack(configPath, nil)
	return node, err
}

// 递归加载配置文件，并把 includes 引入的配置合并到当前配置前面。
// stack 记录当前 include 调用链，用于检测 A includes B、B 又 includes A 这类循环引用。
func loadConfigNodeWithStack(configPath string, stack []string) (*yaml.Node, map[string]string, error) {
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve config path %s: %w", configPath, err)
	}
	absPath = filepath.Clean(absPath)

	// 检查路径，确认是否存在循环引用
	for i, item := range stack {
		if item == absPath {
			// 拷贝一份stack，避免拼接后的cycle切片复用stack切片的底层数组
			// cycle := append(slices.Clone(stack[i:]), absPath)
			cycle := append(stack[i:], absPath)
			return nil, nil, fmt.Errorf("includes cycle: %s", strings.Join(cycle, " -> "))
		}
	}

	slog.Debug(fmt.Sprintf("loading config %s", absPath), "path", absPath)

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read file failed: %w", err)
	}

	root := &yaml.Node{}
	if err := yaml.Unmarshal(content, root); err != nil {
		return nil, nil, fmt.Errorf("unmarshal config failed: %w", err)
	}
	mapping := documentMapping(root)
	if mapping == nil {
		return nil, nil, fmt.Errorf("config %s must be a YAML mapping", absPath)
	}

	// includes 只是加载指令，不应该继续保留在最终合并后的业务配置里。
	includeNodes := removeMappingKeys(mapping, includesKey)
	merged := newMappingNode()
	sources := make(map[string]string)

	for _, includeNode := range includeNodes {
		includePaths, err := resolveIncludePaths(absPath, includeNode)
		if err != nil {
			return nil, nil, err
		}
		for _, includePath := range includePaths {
			// 先合并被 include 的配置，再合并当前文件，让当前文件可以覆盖 include 的默认值。
			included, includedSources, err := loadConfigNodeWithStack(includePath, append(stack, absPath))
			if err != nil {
				return nil, nil, err
			}
			merged, err = mergeMappingNodes(merged, included, sources, includedSources, includePath, nil)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	currentSources := collectSources(mapping, absPath, nil)
	merged, err = mergeMappingNodes(merged, mapping, sources, currentSources, absPath, nil)
	if err != nil {
		return nil, nil, err
	}
	return merged, sources, nil
}

// 解析 includes 字段，支持单个字符串或字符串数组。
// 返回值统一转换为基于当前配置文件目录的绝对/清理后的文件路径列表。
func resolveIncludePaths(baseFile string, includeNode *yaml.Node) ([]string, error) {
	baseDir := filepath.Dir(baseFile)
	var entries []string
	switch includeNode.Kind {
	case yaml.ScalarNode:
		if includeNode.Tag != "!!str" {
			return nil, fmt.Errorf("includes entries must be strings")
		}
		entries = append(entries, includeNode.Value)
	case yaml.SequenceNode:
		for _, item := range includeNode.Content {
			if item.Kind != yaml.ScalarNode || item.Tag != "!!str" {
				return nil, fmt.Errorf("includes entries must be strings")
			}
			entries = append(entries, item.Value)
		}
	default:
		return nil, fmt.Errorf("includes entries must be strings")
	}

	var resolved []string
	for _, entry := range entries {
		paths, err := expandIncludePath(baseDir, entry)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, paths...)
	}
	return resolved, nil
}

// 展开单条 include 路径。普通路径直接返回；带通配符的路径会展开成稳定排序后的匹配列表。
func expandIncludePath(baseDir, includePath string) ([]string, error) {
	pattern := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(includePath)))
	if !hasWildcard(includePath) {
		return []string{pattern}, nil
	}

	var matches []string
	var err error
	if strings.Contains(includePath, "**") {
		// filepath.Glob 不支持 ** 递归匹配，这里单独走 WalkDir + regexp 实现。
		matches, err = expandDoubleStar(baseDir, includePath)
	} else {
		matches, err = filepath.Glob(pattern)
	}
	if err != nil {
		return nil, fmt.Errorf("expand includes entry %s: %w", includePath, err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no files matched includes pattern %q", includePath)
	}
	sortIncludeMatches(matches)
	slog.Debug(fmt.Sprintf("includes pattern %s matched: %s", includePath, strings.Join(matches, ", ")), "pattern", includePath, "matches", matches)
	return matches, nil
}

// 对 include 的通配符结果做稳定排序，避免不同平台或文件系统返回顺序不同。
func sortIncludeMatches(matches []string) {
	sort.Slice(matches, func(i, j int) bool {
		iName := filepath.Base(matches[i])
		jName := filepath.Base(matches[j])
		if iName != jName {
			return iName < jName
		}
		return matches[i] < matches[j]
	})
}

// 展开包含 ** 的 include 模式，按 baseDir 递归遍历所有文件后用正则过滤。
func expandDoubleStar(baseDir, includePath string) ([]string, error) {
	pattern, err := globPatternToRegexp(filepath.ToSlash(includePath))
	if err != nil {
		return nil, err
	}
	var matches []string
	err = filepath.WalkDir(baseDir, func(itemPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(baseDir, itemPath)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if pattern.MatchString(rel) {
			matches = append(matches, itemPath)
		}
		return nil
	})
	return matches, err
}

// 将简化的 glob 模式转换成正则：* 匹配单层路径片段，** 匹配跨目录路径。
func globPatternToRegexp(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				b.WriteString(".*")
				i++
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '{', '}', '^', '$', '[', ']', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// 将 override 合并到 base，并维护每个字段来源文件。
// 顶层单例字段重复会报错；可合并的 mapping 会递归合并；其他字段由后加载的配置覆盖先加载的配置。
func mergeMappingNodes(base, override *yaml.Node, sources map[string]string, overrideSources map[string]string, overrideFile string, pathParts []string) (*yaml.Node, error) {
	if base == nil || base.Kind == 0 {
		base = newMappingNode()
	}
	if override == nil {
		return base, nil
	}
	baseMapping := documentMapping(base)
	overrideMapping := documentMapping(override)
	if baseMapping == nil || overrideMapping == nil {
		return cloneNode(override), nil
	}

	for i := 0; i < len(overrideMapping.Content); i += 2 {
		key := overrideMapping.Content[i]
		value := overrideMapping.Content[i+1]
		if key.Value == includesKey {
			continue
		}
		currentPath := appendPath(pathParts, key.Value)
		baseIndex := findMappingKeyIndex(baseMapping, key.Value)
		if baseIndex < 0 {
			baseMapping.Content = append(baseMapping.Content, cloneNode(key), cloneNode(value))
			copySourcesForPath(sources, overrideSources, currentPath, overrideFile)
			continue
		}

		baseValue := baseMapping.Content[baseIndex+1]
		if len(currentPath) == 1 && isSingletonKey(key.Value) {
			pathKey := strings.Join(currentPath, ".")
			baseSource := sources[pathKey]
			if baseSource == "" {
				baseSource = "earlier config"
			}
			return nil, fmt.Errorf("duplicate singleton field %q from %s already defined in %s", key.Value, overrideFile, baseSource)
		}
		if shouldMergeMapping(currentPath, baseValue, value) {
			mergedChild, err := mergeMappingNodes(baseValue, value, sources, overrideSources, overrideFile, currentPath)
			if err != nil {
				return nil, err
			}
			baseMapping.Content[baseIndex+1] = mergedChild
			continue
		}

		pathKey := strings.Join(currentPath, ".")
		baseSource := sources[pathKey]
		if baseSource == "" {
			baseSource = "earlier config"
		}
		slog.Warn(fmt.Sprintf("warning: key %q from %s overrides value from %s", pathKey, overrideFile, baseSource), "key", pathKey, "override_file", overrideFile, "base_source", baseSource)
		baseMapping.Content[baseIndex+1] = cloneNode(value)
		copySourcesForPath(sources, overrideSources, currentPath, overrideFile)
	}
	return baseMapping, nil
}

// 只合并浅层 mapping：顶层非关键字对象和二级对象可以递归合并，更深层直接整体覆盖。
func shouldMergeMapping(pathParts []string, baseValue, overrideValue *yaml.Node) bool {
	if len(pathParts) == 0 || len(pathParts) > 2 {
		return false
	}
	if len(pathParts) == 1 && IsKeyword(pathParts[0]) {
		return false
	}
	return baseValue.Kind == yaml.MappingNode && overrideValue.Kind == yaml.MappingNode
}

func isSingletonKey(key string) bool {
	_, ok := singletonKeys[key]
	return ok
}

// 收集配置节点中每个字段路径对应的来源文件，用于覆盖日志和重复字段报错。
func collectSources(node *yaml.Node, sourceFile string, pathParts []string) map[string]string {
	sources := make(map[string]string)
	mapping := documentMapping(node)
	if mapping == nil {
		return sources
	}
	for i := 0; i < len(mapping.Content); i += 2 {
		key := mapping.Content[i]
		value := mapping.Content[i+1]
		if key.Value == includesKey {
			continue
		}
		currentPath := appendPath(pathParts, key.Value)
		pathKey := strings.Join(currentPath, ".")
		sources[pathKey] = sourceFile
		if value.Kind == yaml.MappingNode {
			for childPath, childSource := range collectSources(value, sourceFile, currentPath) {
				sources[childPath] = childSource
			}
		}
	}
	return sources
}

// 当某个字段被复制或覆盖时，同步复制其来源信息；没有精确来源时使用 fallback。
func copySourcesForPath(sources map[string]string, overrideSources map[string]string, pathParts []string, fallback string) {
	pathKey := strings.Join(pathParts, ".")
	prefix := pathKey + "."
	used := false
	for key, source := range overrideSources {
		if key == pathKey || strings.HasPrefix(key, prefix) {
			sources[key] = source
			used = true
		}
	}
	if !used {
		sources[pathKey] = fallback
	}
}

// yaml.Unmarshal 会生成 DocumentNode，这里统一取出真正的 mapping 根节点。
func documentMapping(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return documentMapping(node.Content[0])
	}
	if node.Kind == yaml.MappingNode {
		return node
	}
	return nil
}

// 创建空 YAML mapping 节点，作为 include 合并的累积结果容器。
func newMappingNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
}

// 从 mapping 中移除所有指定 key，并按原始 YAML 出现顺序返回被移除的 value 节点。
func removeMappingKeys(mapping *yaml.Node, key string) []*yaml.Node {
	var values []*yaml.Node
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return values
	}
	content := mapping.Content[:0]
	for i := 0; i < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		valueNode := mapping.Content[i+1]
		if keyNode.Value == key {
			values = append(values, valueNode)
			continue
		}
		content = append(content, keyNode, valueNode)
	}
	mapping.Content = content
	return values
}

// YAML mapping 的 Content 以 key/value 交替存储，所以查找时每次前进两个节点。
func findMappingKeyIndex(mapping *yaml.Node, key string) int {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return -1
	}
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return i
		}
	}
	return -1
}

func hasWildcard(value string) bool {
	return strings.ContainsAny(value, "*?[")
}

// 追加路径片段时返回新切片，避免调用方持有的 pathParts 被后续递归修改。
func appendPath(pathParts []string, next string) []string {
	result := make([]string, 0, len(pathParts)+1)
	result = append(result, pathParts...)
	result = append(result, next)
	return result
}

// 深拷贝 YAML 节点，避免合并时修改 include 来源文件解析出的原始节点。
func cloneNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	clone := *node
	if len(node.Content) > 0 {
		clone.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			clone.Content[i] = cloneNode(child)
		}
	}
	return &clone
}
