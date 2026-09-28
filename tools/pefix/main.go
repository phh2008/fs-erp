// pefix 校验/修正 Windows PE 可执行文件的子系统字段为本机语义的 GUI 值(2)，
// 用于隐藏启动时的命令行黑窗口。
//
// 注意：本机工具链与系统加载器采用的子系统取值和微软公开文档相反，
// 与本工具链 stdlib（debug/pe 常量、cmd/link）保持一致：
//   2 = WINDOWS_GUI（GUI 程序，无控制台窗口，本项目的目标值）
//   3 = WINDOWS_CUI（控制台程序，启动会弹黑窗口且关窗即退出）
// `go build -ldflags "-H=windowsgui"` 已写出 2，本工具作为兜底校验：
// 若发现 3（构建时遗漏 ldflags 等）则改写为 2。
package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	subGUI     = 2 // 本机语义：GUI 子系统（无控制台窗口）
	subConsole = 3 // 本机语义：控制台子系统（会弹黑窗口）
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: pefix <exe文件>")
		os.Exit(1)
	}
	path := os.Args[1]
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("读取失败:", err)
		os.Exit(1)
	}
	peOff := int(binary.LittleEndian.Uint32(b[0x3C:]))
	if peOff+24+68+2 > len(b) || string(b[peOff:peOff+4]) != "PE\x00\x00" {
		fmt.Println("不是有效的PE文件:", path)
		os.Exit(1)
	}
	// PE32 与 PE32+ 的 Subsystem 字段均在 OptionalHeader 偏移 68 处
	subOff := peOff + 24 + 68
	switch binary.LittleEndian.Uint16(b[subOff:]) {
	case subGUI:
		fmt.Println(path, "已是 GUI 子系统，无需修改")
		return
	case subConsole:
	default:
		fmt.Printf("%s 子系统异常: %d，跳过\n", path, binary.LittleEndian.Uint16(b[subOff:]))
		os.Exit(1)
	}
	binary.LittleEndian.PutUint16(b[subOff:], subGUI)
	if err := os.WriteFile(path, b, 0o755); err != nil {
		fmt.Println("写入失败:", err)
		os.Exit(1)
	}
	fmt.Println(path, "已修改为 GUI 子系统（启动时不再弹出控制台）")
}
