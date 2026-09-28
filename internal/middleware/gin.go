package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// GinLogger 通过 slog 记录每个 HTTP 请求
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		attrs := []any{
			slog.Int("status", c.Writer.Status()),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("query", query),
			slog.String("ip", c.ClientIP()),
			slog.String("latency", time.Since(start).String()),
		}
		if errMsg := c.Errors.ByType(gin.ErrorTypePrivate).String(); errMsg != "" {
			attrs = append(attrs, slog.String("error", errMsg))
			slog.Error("http request", attrs...)
		} else {
			slog.Info("http request", attrs...)
		}
	}
}
