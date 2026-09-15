// Package buildinfo 保存由 -ldflags -X 注入的构建期元信息。
package buildinfo

import (
	"fmt"
	"runtime"
)

// 以下变量在发布构建时通过 -ldflags "-X ..." 覆盖，默认值用于 go run 开发态。
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String 返回人类可读的版本行，用于 --version 与启动横幅。
func String() string {
	return fmt.Sprintf("zerotier-webui %s (commit %s, built %s, %s)",
		Version, Commit, Date, runtime.Version())
}
