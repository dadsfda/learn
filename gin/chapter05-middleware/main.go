package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// ===== 自定义中间件 =====

// 1. 请求日志中间件
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// 处理请求
		c.Next()

		// 计算耗时
		latency := time.Since(start)
		status := c.Writer.Status()

		log.Printf("[GIN] %s %s | %d | %v | %s",
			method, path, status, latency, c.ClientIP())
	}
}

// 2. 认证中间件
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "请先登录",
			})
			return
		}

		// 模拟 token 验证
		if token != "Bearer my-secret-token" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token 无效",
			})
			return
		}

		// 将用户信息存入上下文
		c.Set("username", "admin")
		c.Next()
	}
}

// 3. CORS 中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// 4. 耗时统计中间件
func Timer() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		c.Header("X-Response-Time", latency.String())
	}
}

func main() {
	f, _ := os.Create("gin.log")
	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)
	
	r := gin.New()

	// 全局中间件
	r.Use(gin.Recovery())  // 恢复 panic
	r.Use(RequestLogger()) // 请求日志
	r.Use(CORS())          // 跨域
	r.Use(Timer())         // 耗时统计

	// 公开路由（不需要认证）
	public := r.Group("/api")
	{
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		public.POST("/login", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"token":   "my-secret-token",
				"message": "登录成功",
			})
		})
	}

	// 需要认证的路由
	protected := r.Group("/api", AuthRequired())
	{
		protected.GET("/profile", func(c *gin.Context) {
			username, _ := c.Get("username")
			c.JSON(200, gin.H{
				"username": username,
				"message":  "这是受保护的接口",
			})
		})

		protected.PUT("/profile", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "更新成功"})
		})
	}

	fmt.Println("服务器启动在 http://localhost:8080")
	fmt.Println("测试认证:")
	fmt.Println("  curl http://localhost:8080/api/profile -H \"Authorization: Bearer my-secret-token\"")
	r.Run()
}
