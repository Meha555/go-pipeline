package main

import (
	"fmt"
	"go-pipeline/info"
	"go-pipeline/job"
	"go-pipeline/job/build"
	"go-pipeline/job/deploy"
	"go-pipeline/job/test"
	"go-pipeline/pipeline"
	"go-pipeline/stage"
)

func main() {
	srcInfo := info.SourceInfo{
		BranchName: "release_v1.0",
	}

	pipeline := pipeline.NewPipeline("一次构建流程").
		AddStage(*stage.NewStage("说明").
			AddJob(job.NewJob("一次构建说明", func(args ...interface{}) bool {
				if len(args) > 0 {
					if branchName, ok := args[0].(string); ok {
						fmt.Printf("构建版本: %s\n", branchName)
						return true
					}
				}
				return false
			}, srcInfo.BranchName.ValueOr("未命名")))).
		AddStage(*stage.NewStage("构建").
			AddJob(job.NewJob("获取编译代码", build.Pull, srcInfo)).
			AddJob(job.NewJob("代码静态分析", build.Check, srcInfo)).
			AddJob(job.NewJob("代码构建", build.Build, srcInfo)).
			AddJob(job.NewJob("推送构建结果", build.Push, srcInfo))).
		AddStage(*stage.NewStage("测试").
			AddJob(job.NewJob("测试工程: X-59", test.Test, 2021, "X-59: QueSST 洛克希德·马丁", srcInfo.PushId)).
			AddJob(job.NewJob("测试工程: CH-47", test.Test, 1956, "CH-47: Chinook 波音", srcInfo.PushId))).
		AddStage(*stage.NewStage("部署").
			AddJob(job.NewJob("部署环境: 北京", deploy.Deploy, "清华大学", srcInfo.PushId)).
			AddJob(job.NewJob("部署环境: 上海", deploy.Deploy, "复旦大学", srcInfo.PushId)).
			AddJob(job.NewJob("部署环境: 深圳", deploy.Deploy, "深圳大学", srcInfo.PushId)))
	if pipeline.Run() {
		fmt.Println("流水线执行成功")
	} else {
		fmt.Println("流水线执行失败")
	}
}
