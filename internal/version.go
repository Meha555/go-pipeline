package internal

import "runtime/debug"

var Version string

func ResolveVersion(buildVersion string) string {
	if Version != "" && Version != "(devel)" {
		return Version
	}
	if buildVersion != "" && buildVersion != "(devel)" {
		return buildVersion
	}
	return "dev"
}

func BuildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Version
}
