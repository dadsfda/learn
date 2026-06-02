package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// 模拟慢请求
	r.GET("/slow", func(c *gin.Context) {
		time.Sleep(3 * time.Second)
		c.JSON(200, gin.H{"message": "slow response done"})
	})

	// Cookie 操作
	r.GET("/set-cookie", func(c *gin.Context) {
		c.SetCookie("session_id", "abc123", 3600, "/", "localhost", false, true)
		c.JSON(200, gin.H{"message": "Cookie 已设置"})
	})

	r.GET("/get-cookie", func(c *gin.Context) {
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			c.JSON(400, gin.H{"error": "Cookie 不存在"})
			return
		}
		c.JSON(200, gin.H{"session_id": sessionID})
	})

	// 创建自定义 HTTP 服务器
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// 启动服务器（非阻塞）
	go func() {
		log.Println("服务器启动在 http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器错误: %v", err)
		}
	}()

	// 等待中断信号（Ctrl+C 或 kill）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("收到关闭信号，正在优雅关闭...")

	// 给 5 秒时间处理剩余请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("服务器关闭失败: %v", err)
	}

	log.Println("服务器已安全关闭")
}
