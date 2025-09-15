package internal

type ContextKey string

const (
	VerboseKey ContextKey = "verbose"
	TraceKey   ContextKey = "trace"
	DryRunKey  ContextKey = "dry-run"
)