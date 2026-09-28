package main

import (
	"fmt"
	"log/slog"
	_ "embed"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fs-erp/internal/autostart"
	"github.com/fs-erp/internal/database"
	"github.com/fs-erp/internal/handler"
	"github.com/fs-erp/internal/logger"
	"github.com/fs-erp/internal/middleware"
	"github.com/getlantern/systray"
	"github.com/gin-gonic/gin"
)

const addr = "0.0.0.0:51873" // 监听所有网卡，同网段设备可通过 本机IP:51873 访问
const hostURL = "http://127.0.0.1:51873" // 本机访问入口，自动打开页面的地址

//go:embed web/index.html
var indexHTML []byte

//go:embed web/icon.ico
var iconICO []byte

func main() {
	// 命令行子命令（--autostart 等），处理后退出
	if len(os.Args) > 1 {
		// 仅子命令模式附着控制台输出。无参数直接启动（双击/自启）不调用，
		// 避免给 GUI 进程引入多余的控制台行为
		attachParentConsole()
		switch os.Args[1] {
		case "--autostart":
			handleAutostart(os.Args[2:])
			return
		case "-h", "--help":
			printUsage()
			return
		}
	}

	dataDir := resolveDataDir()
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		slog.Error("创建数据目录失败", "dir", dataDir, "error", err)
		panic(err)
	}

	// 日志初始化：slog + zap + lumberjack（滚动文件，默认配置见 logger.DefaultConfig）
	logCfg := logger.DefaultConfig()
	logCfg.Filename = filepath.Join(dataDir, "logs", "fs-erp.log")
	logger.Init(logCfg)
	slog.Info("数据目录", "dir", dataDir)

	db := database.Init(filepath.Join(dataDir, "fs-erp.db"))
	goodsHandler := handler.NewGoodsHandler(db)
	brandHandler := handler.NewBrandHandler(db)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.GinLogger(), gin.Recovery())

	// 主页（页面已嵌入 exe，单文件发布）
	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
	// 网站图标（与托盘同一份嵌入图标）
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Data(http.StatusOK, "image/x-icon", iconICO)
	})

	api := r.Group("/api")
	{
		// 商品
		api.GET("/goods", goodsHandler.ListGoods)                // 分页查询
		api.POST("/goods", goodsHandler.CreateGoods)             // 新增
		api.PUT("/goods/:id", goodsHandler.UpdateGoods)          // 编辑
		api.PATCH("/goods/:id/stock", goodsHandler.UpdateStock)  // 行内改库存
		api.DELETE("/goods/:id", goodsHandler.DeleteGoods)       // 删除（逻辑删除）
		api.GET("/goods/export", goodsHandler.ExportGoods)       // 导出Excel
		api.POST("/goods/import", goodsHandler.ImportGoods)      // 导入Excel
		api.GET("/goods/template", goodsHandler.ImportTemplate)  // 导入模板

		// 品牌
		api.GET("/brands", brandHandler.ListBrands)
		api.POST("/brands", brandHandler.CreateBrand)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// 端口被占用：可能已有实例在运行，直接打开其页面后退出
		slog.Warn("端口被占用，可能服务已在运行，将打开已有实例页面", "addr", addr, "error", err)
		openBrowser(hostURL)
		return
	}
	url := hostURL
	slog.Info("商品管理系统已启动",
		"addr", addr, "url", url,
		"局域网访问", "http://<本机IP>:51873")

	// HTTP 服务放后台 goroutine，主协程运行托盘
	go func() {
		if err := r.RunListener(ln); err != nil {
			slog.Error("服务运行异常", "error", err)
			os.Exit(1)
		}
	}()
	go openBrowser(url)

	systray.Run(trayOnReady, func() {
		slog.Info("服务已退出")
	})
}

// trayOnReady 初始化系统托盘图标与菜单
func trayOnReady() {
	systray.SetIcon(iconICO)
	systray.SetTitle("商品管理系统")
	systray.SetTooltip("商品管理系统运行中: " + hostURL)
	mOpen := systray.AddMenuItem("打开页面", "在浏览器中打开商品管理系统")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "停止服务并退出程序")
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				openBrowser(hostURL)
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

// printUsage 打印命令行用法
func printUsage() {
	fmt.Println(`商品管理系统

用法:
  fs-erp.exe                          启动服务并打开浏览器
  fs-erp.exe --autostart install      设置开机自启（登录后自动启动）
  fs-erp.exe --autostart uninstall    取消开机自启
  fs-erp.exe --autostart status       查看自启状态
  fs-erp.exe -h                       显示帮助

数据目录: %AppData%\fs-erp（可通过环境变量 FS_ERP_DATA_DIR 自定义）`)
}

// handleAutostart 处理开机自启子命令
func handleAutostart(args []string) {
	sub := "status"
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "install":
		if err := autostart.Install(); err != nil {
			fmt.Println("设置开机自启失败:", err)
			os.Exit(1)
		}
		fmt.Println("已设置开机自启，下次登录后自动启动")
	case "uninstall":
		if err := autostart.Uninstall(); err != nil {
			fmt.Println("取消开机自启失败:", err)
			os.Exit(1)
		}
		fmt.Println("已取消开机自启")
	case "status":
		enabled, path, err := autostart.Status()
		if err != nil {
			fmt.Println("查询自启状态失败:", err)
			os.Exit(1)
		}
		if enabled {
			fmt.Println("开机自启: 已启用")
			fmt.Println("启动命令:", path)
		} else {
			fmt.Println("开机自启: 未启用")
		}
	default:
		fmt.Println("未知子命令:", sub)
		printUsage()
		os.Exit(1)
	}
}

// resolveDataDir 解析数据存储目录（数据库、日志）：
// 优先环境变量 FS_ERP_DATA_DIR，否则使用用户数据目录 %AppData%\fs-erp，
// 避免数据文件生成在 exe 同目录被误删
func resolveDataDir() string {
	if dir := os.Getenv("FS_ERP_DATA_DIR"); dir != "" {
		return dir
	}
	if cfg, err := os.UserConfigDir(); err == nil {
		return filepath.Join(cfg, "fs-erp")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".fs-erp")
	}
	return "data"
}
