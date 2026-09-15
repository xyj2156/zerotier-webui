// Package web 内置 Vite 构建产物（dist/），让前端与后端打进同一个可执行文件。
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedded embed.FS

// Dist 返回以 dist/ 为根的只读文件系统，供 HTTP 层托管前端产物。
func Dist() fs.FS {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		// embed 路径写错属于编码期问题，崩掉比静默 404 更易发现。
		panic("无法定位嵌入的 dist 目录：" + err.Error())
	}
	return sub
}

// Built 报告二进制里是否含真实前端产物（而非仅占位文件）。
func Built() bool {
	info, err := fs.Stat(embedded, "dist/index.html")
	return err == nil && !info.IsDir()
}
