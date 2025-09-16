package internal

type ContextKey string

const (
	VerboseKey   ContextKey = "verbose"
	NoSilenceKey ContextKey = "no-silence"
	TraceKey     ContextKey = "trace"
	DryRunKey    ContextKey = "dry-run"
)
