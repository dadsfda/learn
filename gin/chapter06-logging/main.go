package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 日志输出到文件 + 控制台
	f, _ := os.Create("gin.log")
	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)

	// 2. 禁用颜色（生产环境）
	// gin.DisableConsoleColor()

	r := gin.New()

	// 3. 自定义日志格式
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %s %d %v\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Method,
			param.Path,
			param.ClientIP,
			param.StatusCode,
			param.Latency,
		)
	}))

	r.Use(gin.Recovery())

	// 4. 结构化日志中间件（使用 slog）
	structuredLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()

		structuredLogger.Info("request completed",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency", time.Since(start).String(),
			"client_ip", c.ClientIP(),
		)
	})

	// 路由
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	r.GET("/slow", func(c *gin.Context) {
		time.Sleep(100 * time.Millisecond)
		c.JSON(200, gin.H{"message": "slow response"})
	})

	r.Run()
}
