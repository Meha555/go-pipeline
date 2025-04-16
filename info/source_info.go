package info

import (
	"fmt"
	"go-pipeline/common"
	"strings"
)

// SourceInfo 表示源码信息
type SourceInfo struct {
	BranchName common.OptString // 创建时产生
	CommitId   common.OptString // pull时产生
	Date       common.OptString // pull时产生
	BuildId    common.OptString // build时产生
	PushId     common.OptString // push时产生
}

// CoutSourceInfo 输出源码信息
func (s SourceInfo) String(sourceInfo SourceInfo) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("分支信息: %s\n", sourceInfo.BranchName.ValueOr("未指定")))
	builder.WriteString(fmt.Sprintf("提交标识: %s\n", sourceInfo.CommitId.ValueOr("未指定")))
	builder.WriteString(fmt.Sprintf("提交日期: %s\n", sourceInfo.Date.ValueOr("未指定")))
	builder.WriteString(fmt.Sprintf("构建标识: %s\n", sourceInfo.BuildId.ValueOr("未指定")))
	builder.WriteString(fmt.Sprintf("推送标识: %s\n", sourceInfo.PushId.ValueOr("未指定")))
	return builder.String()
}
