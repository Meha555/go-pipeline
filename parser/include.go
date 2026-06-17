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

const includeKey = "include"

// 仅允许出现一次的关键字
var singletonKeys = map[string]struct{}{
	keywordName:    {},
	keywordVersion: {},
	keywordShell:   {},
	keyWordCron:    {},
	keywordWorkdir: {},
}

func loadConfigNode(configPath string) (*yaml.Node, error) {
	node, _, err := loadConfigNodeWithStack(configPath, nil)
	return node, err
}

func loadConfigNodeWithStack(configPath string, stack []string) (*yaml.Node, map[string]string, error) {
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve config path %s: %w", configPath, err)
	}
	absPath = filepath.Clean(absPath)

	for i, item := range stack {
		if item == absPath {
			cycle := append(append([]string{}, stack[i:]...), absPath)
			return nil, nil, fmt.Errorf("include cycle: %s", strings.Join(cycle, " -> "))
		}
	}

	if _, err := os.Stat(absPath); err != nil {
		return nil, nil, err
	}
	slog.Debug(fmt.Sprintf("load config %s", absPath), "path", absPath)

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

	includeNode := removeMappingKey(mapping, includeKey)
	merged := newMappingNode()
	sources := make(map[string]string)

	if includeNode != nil {
		includePaths, err := resolveIncludePaths(absPath, includeNode)
		if err != nil {
			return nil, nil, err
		}
		for _, includePath := range includePaths {
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

func resolveIncludePaths(baseFile string, includeNode *yaml.Node) ([]string, error) {
	baseDir := filepath.Dir(baseFile)
	var entries []string
	switch includeNode.Kind {
	case yaml.ScalarNode:
		if includeNode.Tag != "!!str" {
			return nil, fmt.Errorf("include entries must be strings")
		}
		entries = append(entries, includeNode.Value)
	case yaml.SequenceNode:
		for _, item := range includeNode.Content {
			if item.Kind != yaml.ScalarNode || item.Tag != "!!str" {
				return nil, fmt.Errorf("include entries must be strings")
			}
			entries = append(entries, item.Value)
		}
	default:
		return nil, fmt.Errorf("include entries must be strings")
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

func expandIncludePath(baseDir, includePath string) ([]string, error) {
	pattern := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(includePath)))
	if !hasWildcard(includePath) {
		return []string{pattern}, nil
	}

	var matches []string
	var err error
	if strings.Contains(includePath, "**") {
		matches, err = expandDoubleStar(baseDir, includePath)
	} else {
		matches, err = filepath.Glob(pattern)
	}
	if err != nil {
		return nil, fmt.Errorf("expand include %s: %w", includePath, err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no files matched include pattern %q", includePath)
	}
	sortIncludeMatches(matches)
	slog.Debug(fmt.Sprintf("include pattern %s matched: %s", includePath, strings.Join(matches, ", ")), "pattern", includePath, "matches", matches)
	return matches, nil
}

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
		if key.Value == includeKey {
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

func collectSources(node *yaml.Node, sourceFile string, pathParts []string) map[string]string {
	sources := make(map[string]string)
	mapping := documentMapping(node)
	if mapping == nil {
		return sources
	}
	for i := 0; i < len(mapping.Content); i += 2 {
		key := mapping.Content[i]
		value := mapping.Content[i+1]
		if key.Value == includeKey {
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

func newMappingNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
}

func removeMappingKey(mapping *yaml.Node, key string) *yaml.Node {
	idx := findMappingKeyIndex(mapping, key)
	if idx < 0 {
		return nil
	}
	value := mapping.Content[idx+1]
	mapping.Content = append(mapping.Content[:idx], mapping.Content[idx+2:]...)
	return value
}

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

func appendPath(pathParts []string, next string) []string {
	result := make([]string, 0, len(pathParts)+1)
	result = append(result, pathParts...)
	result = append(result, next)
	return result
}

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
