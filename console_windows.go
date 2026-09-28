//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// attachParentConsole GUI 程序从命令行(cmd/PowerShell)运行子命令时，
// 重新附着父控制台，使 --autostart 等子命令的输出可见。
// 注意：只能在子命令模式下调用（main 中仅在带参数时调用），
// 无参数直接启动（双击/自启）不引入任何控制台行为。
func attachParentConsole() {
	// 已有有效标准输出（重定向/管道等）时不附着，保持输出走向不变
	if h, _ := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE); h != 0 && h != windows.InvalidHandle {
		return
	}
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	attach := kernel32.NewProc("AttachConsole")
	r, _, _ := attach.Call(^uintptr(0)) // ATTACH_PARENT_PROCESS = (DWORD)-1
	if r == 0 {
		return // 没有可附着的父控制台
	}
	if conout, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
		os.Stdout = conout
		os.Stderr = conout
	}
	if conin, err := os.OpenFile("CONIN$", os.O_RDONLY, 0); err == nil {
		os.Stdin = conin
	}
}
