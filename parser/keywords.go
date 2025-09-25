package parser

import "slices"

// keywords
const (
	keywordName    = "name"
	keywordVersion = "version"
	keywordShell   = "shell"

	keywordEnvs    = "envs"
	keywordWorkdir = "workdir"

	keywordStages = "stages"

	keywordJobs         = "jobs"
	keywordStage        = "stage"
	keywordActions      = "actions"
	keywordTimeout      = "timeout"
	keywordAllowFailure = "allow_failure"
	keywordSkips        = "skips"
	keywordHooks        = "hooks"
	keywordHookBefore   = "before"
	keywordHookAfter    = "after"
)

var keywordMap = []string{
	keywordName,
	keywordVersion,
	keywordShell,
	keywordEnvs,
	keywordWorkdir,
	keywordStages,
	keywordJobs,
	keywordStage,
	keywordActions,
	keywordTimeout,
	keywordAllowFailure,
	keywordSkips,
	keywordHooks,
	keywordHookBefore,
	keywordHookAfter,
}

func IsKeyword(token string) bool {
	return slices.Contains(keywordMap, token)
}
