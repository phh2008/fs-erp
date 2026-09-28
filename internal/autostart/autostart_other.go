//go:build !windows

package autostart

import "fmt"

// Install 非 Windows 平台不支持注册表自启
func Install() error {
	return fmt.Errorf("开机自启仅支持 Windows 平台")
}

// Uninstall 非 Windows 平台不支持注册表自启
func Uninstall() error {
	return fmt.Errorf("开机自启仅支持 Windows 平台")
}

// Status 非 Windows 平台不支持注册表自启
func Status() (bool, string, error) {
	return false, "", fmt.Errorf("开机自启仅支持 Windows 平台")
}
