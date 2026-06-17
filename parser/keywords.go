package parser

import "slices"

// keywords
const (
	keywordName    = "name"
	keywordVersion = "version"
	keywordShell   = "shell"
	keyWordCron    = "cron"
	keywordInclude = "include"

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
	keywordInclude,
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
	keywordSkips,
	keywordHooks,
	keywordHookBefore,
	keywordHookAfter,
}

func IsKeyword(token string) bool {
	return slices.Contains(keywordMap, token)
}
