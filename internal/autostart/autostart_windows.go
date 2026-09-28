//go:build windows

package autostart

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const valueName = "fs-erp"

// exePath 当前可执行文件绝对路径
func exePath() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(p)
}

// Install 设置开机自启（写入 HKCU Run 键，登录后自动启动，无需管理员权限）
func Install() error {
	exe, err := exePath()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %w", err)
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开注册表失败: %w", err)
	}
	defer k.Close()
	if err := k.SetStringValue(valueName, `"`+exe+`"`); err != nil {
		return fmt.Errorf("写入注册表失败: %w", err)
	}
	return nil
}

// Uninstall 取消开机自启
func Uninstall() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开注册表失败: %w", err)
	}
	defer k.Close()
	if err := k.DeleteValue(valueName); err != nil {
		if err == registry.ErrNotExist {
			return nil // 本来就没有，视为成功
		}
		return fmt.Errorf("删除注册表项失败: %w", err)
	}
	return nil
}

// Status 查看自启状态，返回 是否启用、注册的启动命令
func Status() (bool, string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false, "", fmt.Errorf("打开注册表失败: %w", err)
	}
	defer k.Close()
	val, _, err := k.GetStringValue(valueName)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, "", nil
		}
		return false, "", fmt.Errorf("读取注册表失败: %w", err)
	}
	return true, val, nil
}
