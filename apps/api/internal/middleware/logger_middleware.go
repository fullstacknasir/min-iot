package middleware

import (
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var Logger *slog.Logger

func InitLogger(env string) {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}

// StructuredLogger returns a Gin middleware logging HTTP transactions as structured log events.
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		orgID, _ := c.Get("organization_id")
		userID, _ := c.Get("user_id")

		attrs := []slog.Attr{
			slog.Int("status", statusCode),
			slog.String("method", method),
			slog.String("path", path),
			slog.String("query", rawQuery),
			slog.String("ip", clientIP),
			slog.Float64("latency_ms", float64(latency.Microseconds())/1000.0),
		}

		if orgID != nil && orgID != "" {
			attrs = append(attrs, slog.Any("org_id", orgID))
		}
		if userID != nil && userID != "" {
			attrs = append(attrs, slog.Any("user_id", userID))
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("error", c.Errors.String()))
		}

		ctx := c.Request.Context()
		switch {
		case statusCode >= 500:
			Logger.LogAttrs(ctx, slog.LevelError, "HTTP Server Error", attrs...)
		case statusCode >= 400:
			Logger.LogAttrs(ctx, slog.LevelWarn, "HTTP Client Warning", attrs...)
		default:
			Logger.LogAttrs(ctx, slog.LevelInfo, "HTTP Request Completed", attrs...)
		}
	}
}
