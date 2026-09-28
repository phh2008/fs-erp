package database

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fs-erp/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

const backupKeep = 10 // 保留的备份数量

// Init 初始化 SQLite 数据库（内嵌，纯 Go 驱动，无需 CGO）
// dbPath 为绝对路径时数据与程序目录分离，避免用户误删
func Init(dbPath string) *gorm.DB {
	backupDB(dbPath)
	// WAL 模式 + 忙等待超时，提升批量写入性能
	dsn := "file:" + dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("打开数据库失败", "path", dbPath, "error", err)
		panic(err)
	}
	if err := db.AutoMigrate(&model.Goods{}, &model.Brand{}); err != nil {
		slog.Error("数据库迁移失败", "error", err)
		panic(err)
	}
	seed(db)
	return db
}

// backupDB 启动时备份数据库到 backups 目录，仅保留最近 backupKeep 份
func backupDB(dbPath string) {
	if _, err := os.Stat(dbPath); err != nil {
		return // 首次启动无数据可备份
	}
	backupDir := filepath.Join(filepath.Dir(dbPath), "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		slog.Warn("创建备份目录失败", "dir", backupDir, "error", err)
		return
	}

	name := "fs-erp-" + time.Now().Format("20060102-150405") + ".bak"
	if err := copyFile(dbPath, filepath.Join(backupDir, name)); err != nil {
		slog.Warn("数据库备份失败", "error", err)
		return
	}
	// WAL 模式下同时备份 -wal/-shm 文件，避免未合并事务丢失
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(dbPath + suffix); err == nil {
			_ = copyFile(dbPath+suffix, filepath.Join(backupDir, name)+suffix)
		}
	}
	slog.Info("数据库已备份", "file", name)

	// 清理旧备份（文件名含时间戳，按名称排序即按时间排序）
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return
	}
	var baks []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".bak") {
			baks = append(baks, e.Name())
		}
	}
	sort.Strings(baks)
	for i := 0; i < len(baks)-backupKeep; i++ {
		_ = os.Remove(filepath.Join(backupDir, baks[i]))
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// seed 初始化品牌枚举数据
func seed(db *gorm.DB) {
	var count int64
	db.Model(&model.Brand{}).Count(&count)
	if count > 0 {
		return
	}
	now := model.LocalTime(time.Now())
	brands := []model.Brand{
		{Name: "Apple", CreateTime: now},
		{Name: "华为", CreateTime: now},
		{Name: "小米", CreateTime: now},
		{Name: "三星", CreateTime: now},
		{Name: "其他", CreateTime: now},
	}
	if err := db.Create(&brands).Error; err != nil {
		slog.Error("初始化品牌数据失败", "error", err)
	}
}
