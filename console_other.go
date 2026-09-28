//go:build !windows

package main

// attachParentConsole 非 Windows 平台无需处理
func attachParentConsole() {}
