package parser

import "slices"

// keywords
const (
	keywordName     = "name"
	keywordVersion  = "version"
	keywordShell    = "shell"
	keyWordCron     = "cron"
	keywordIncludes = "includes"

	keywordNotifiers = "notifiers"

	keywordEnvs    = "envs"
	keywordWorkdir = "workdir"

	keywordStages = "stages"

	keywordJobs         = "jobs"
	keywordStage        = "stage"
	keywordActions      = "actions"
	keywordTimeout      = "timeout"
	keywordAllowFailure = "allow_failure"
	keywordExports      = "exports"
	keywordRules        = "rules"
	keywordOn           = "on"
	keywordSkips        = "skips"
	keywordHooks        = "hooks"
	keywordHookBefore   = "before"
	keywordHookAfter    = "after"
)

var keywordMap = []string{
	keywordName,
	keywordVersion,
	keywordShell,
	keyWordCron,
	keywordIncludes,
	keywordNotifiers,
	keywordEnvs,
	keywordWorkdir,
	keywordStages,
	keywordJobs,
	keywordStage,
	keywordActions,
	keywordTimeout,
	keywordAllowFailure,
	keywordExports,
	keywordRules,
	keywordOn,
	keywordSkips,
	keywordHooks,
	keywordHookBefore,
	keywordHookAfter,
}

func IsKeyword(token string) bool {
	return slices.Contains(keywordMap, token)
}

// 仅允许出现一次的关键字
var singletonKeys = map[string]struct{}{
	keywordName:    {},
	keywordVersion: {},
	keywordShell:   {},
	keyWordCron:    {},
	keywordWorkdir: {},
	keywordStages:  {},
}

func isMergeableKey(key string) bool {
	return !isSingletonKey(key)
}
