//go:build !windows

package main

import (
	"log/slog"
	"os/exec"
)

// openBrowser 打开默认浏览器（非 Windows 平台，仅保持可编译）
func openBrowser(url string) {
	cmd := exec.Command("xdg-open", url)
	if err := cmd.Start(); err != nil {
		slog.Error("打开浏览器失败，请手动访问", "url", url)
	}
}
