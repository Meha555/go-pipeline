package build

import (
	"go-pipeline/common/cout"
	"go-pipeline/info"
)

func Build(args ...interface{}) bool {
	cout.ElapsedSecondsDummyJob(30, 20)

	// build(job.Args)
	// return job.PostAction()
	return true
}

func build(sourceInfo info.SourceInfo) bool {
	return true
}
