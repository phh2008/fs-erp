//go:build windows

package main

import (
	"log/slog"

	"golang.org/x/sys/windows"
)

// openBrowser 用 ShellExecuteW 直接唤起系统默认浏览器。
// 不再通过 rundll32 子进程打开：rundll32 是控制台子系统程序，
// 由 GUI 进程启动它会额外闪现一个控制台黑窗口。
func openBrowser(url string) {
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		slog.Error("打开浏览器失败，请手动访问", "url", url)
		return
	}
	target, err := windows.UTF16PtrFromString(url)
	if err != nil {
		slog.Error("打开浏览器失败，请手动访问", "url", url)
		return
	}
	if err := windows.ShellExecute(0, verb, target, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		slog.Error("打开浏览器失败，请手动访问", "url", url)
	}
}
