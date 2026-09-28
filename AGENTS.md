# AGENTS.md

This file provides guidance to Agent Code when working with code in this repository.

## 项目概述

fs-erp 是一个 Windows 单文件桌面商品管理系统：Gin HTTP 服务 + 系统托盘常驻，启动后自动打开浏览器（http://127.0.0.1:51873），无登录认证。页面与图标经 `go:embed` 嵌入 exe，发布时只发一个 exe 文件。

技术栈：Gin + GORM + glebarez/sqlite（纯 Go SQLite 驱动，无 CGO）+ excelize（Excel 导入导出）+ zap/lumberjack（日志滚动）+ getlantern/systray（托盘）。

## 常用命令

```bash
make build        # 编译单文件 fs-erp.exe（含 PE 子系统补丁，GUI 模式无控制台窗口）
make run          # 开发模式 go run .
make check        # go vet ./...
make release      # 交叉编译 Windows amd64/arm64 到 dist/（文件名带版本时间戳）
make clean        # 清理产物
make autostart-install / uninstall / status   # 开机自启（写 HKCU Run 注册表键）
```

- 运行测试：本项目暂无测试文件；如有，用 `go test ./...`，单个测试 `go test -run TestName ./internal/xxx/`
- **没有 `go test` 之外的 lint 配置**，`make check` 是唯一静态检查

### 构建关键注意事项

- **页面是嵌入 exe 的**（`//go:embed web/index.html`）：修改 `web/` 下任何文件后必须 `make build` 并**重启 fs-erp.exe** 才生效，浏览器强刷（Ctrl+F5）无法解决旧 exe 的问题
- **PE 子系统取值与微软公开文档相反**：本机工具链（go1.27.1）与系统加载器实际采用 `2=GUI（无控制台）、3=控制台（弹黑窗且关窗即退出）`，与 stdlib `debug/pe`/`cmd/link` 常量一致。`-ldflags "-H=windowsgui"` 会写出 2，即隐藏控制台的正确方式；`tools/pefix` 据此在构建后把 3 兜底改写为 2（`make build`/`make release` 已自动执行）。**切勿按微软标准把 2 改写成 3**，那会让程序变回控制台程序（黑窗口、关窗即退出）
- Linux/macOS 交叉编译会失败：systray 在非 Windows 平台依赖 CGO；release 仅编译 Windows 双架构
- exe 运行中会占用 51873 端口；重复启动时程序自动打开已运行实例的页面后退出（main.go 端口占用兜底逻辑），测试时注意先退出托盘里的旧实例

## 架构

```
main.go                     入口：CLI子命令分发(--autostart/-h)、数据目录解析、
                            路由注册、HTTP 服务后台 goroutine + 主协程托盘消息循环
internal/model/model.go     Goods(gds_goods)、Brand(sys_brand)、LocalTime(JSON格式化/DB读写)
internal/database/          SQLite 初始化：启动前备份(backups/保留10份,含-wal/-shm)、
                            AutoMigrate、品牌种子数据
internal/handler/goods.go   商品 CRUD/分页/行内库存/导入/导出/模板
internal/handler/brand.go   品牌枚举的增查
internal/logger/            slog→zap 桥接 handler + lumberjack 滚动（text/json、级别可配）
internal/middleware/gin.go  请求日志中间件（经 slog 输出）
internal/autostart/         HKCU Run 注册表键读写（windows/其他平台分文件构建）
tools/pefix/                构建后 PE 子系统补丁工具
web/index.html              唯一前端页面（原生 JS 单页，无构建步骤）
```

### 数据与业务规则

- **数据落盘位置**：`%AppData%\fs-erp`（db + logs/ + backups/），环境变量 `FS_ERP_DATA_DIR` 可覆盖；SQLite 开启 WAL 模式（DSN pragma）
- **品牌存名称不存 ID**：`gds_goods.brand` 保存品牌名称字符串（与历史数据一致），`sys_brand` 仅作下拉枚举数据源；导入/保存遇到新品牌会自动写入品牌表
- **商品唯一性**：货号+名称+品牌组合唯一；Excel 导入时存在则覆盖、不存在则新增，文件内重复行合并、空行忽略、库存为空默认 100
- **删除是逻辑删除**（`deleted` 字段 0/1），所有查询需带 `deleted = 0` 条件
- 导入性能依赖批量写路径：一次预加载现有商品建内存索引 + 单事务 `CreateInBatches`（8000 行约 0.6s），不要改回逐行 SELECT/INSERT

### 已知坑

- GORM tag 里不要写 `unsigned`（如 `type:tinyint(1) unsigned`），SQLite 会迁移报错
- GUI 进程里不要调用 `AttachConsole`（除非确需控制台输出，如子命令模式）：Win11 默认终端为 Windows Terminal 时，会给进程分配一个控制台黑窗口。`main.go` 仅在带命令行参数时才调用 `attachParentConsole`，直接启动不能调用
- GUI 进程里不要用 `exec.Command` 启动控制台子系统程序（如 `rundll32`、`netsh`）：Windows 会为其分配控制台黑窗口。openBrowser 在 Windows 走 `browser_windows.go` 的 `ShellExecuteW` 直接唤起浏览器，就是这个原因；若未来需要调用命令行工具，必须设 `SysProcAttr{HideWindow: true, CreationFlags: CREATE_NO_WINDOW}`
- 前端 JS 改动后可用 `node` 校验语法：整页 script 语法错误会导致所有按钮 onclick 静默失效
- 表格列宽用 `<colgroup><col>` 控制宽度（table-layout: fixed），单元格内容需 `max-width: 100%` 跟随列宽，写死 px 会导致拖拽列宽时内容不变
