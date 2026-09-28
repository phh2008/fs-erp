package logger

import (
	"log/slog"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config 日志配置
type Config struct {
	Level      string // 日志级别：debug/info/warn/error
	Format     string // 输出格式：text/json
	MaxSize    int    // 单个日志文件最大体积（MB），超过后滚动
	MaxBackups int    // 保留的旧日志文件最大数量
	MaxAge     int    // 旧日志文件保留天数
	Filename   string // 日志文件路径
}

// DefaultConfig 默认配置：max_size:100、max_backups:10、max_age:30、level:debug、format:text
func DefaultConfig() Config {
	return Config{
		Level:      "debug",
		Format:     "text",
		MaxSize:    100,
		MaxBackups: 10,
		MaxAge:     30,
		Filename:   "logs/fs-erp.log",
	}
}

// Init 初始化日志：zap 核心 + lumberjack 文件滚动 + slog 统一接口。
// 日志同时输出到控制台和滚动文件，并设置为 slog 默认 logger。
func Init(cfg Config) {
	level := parseLevel(cfg.Level)

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}

	// lumberjack 文件滚动
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		LocalTime:  true,
		Compress:   false,
	})
	writer := zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), fileWriter)

	core := zapcore.NewCore(encoder, writer, zap.NewAtomicLevelAt(zapLevelOf(level)))
	zapLogger := zap.New(core, zap.AddCallerSkip(1))

	slog.SetDefault(slog.New(newZapHandler(zapLogger, level)))
}

func parseLevel(s string) slog.Level {
	switch s {
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}
