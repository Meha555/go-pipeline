package parser

import "slices"

// keywords
const (
	keywordName    = "name"
	keywordVersion = "version"
	keywordEnvs    = "envs"
	keywordWorkdir = "workdir"

	keywordStages = "stages"

	keywordJobs         = "jobs"
	keywordStage        = "stage"
	keywordActions      = "actions"
	keywordTimeout      = "timeout"
	keywordAllowFailure = "allow_failure"
)

var keywordMap = []string{
	keywordName,
	keywordVersion,
	keywordEnvs,
	keywordWorkdir,
	keywordStages,
	keywordJobs,
	keywordStage,
	keywordActions,
	keywordTimeout,
	keywordAllowFailure,
}

func IsKeyword(token string) bool {
	return slices.Contains(keywordMap, token)
}
