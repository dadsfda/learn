package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// 创建带有默认中间件（Logger + Recovery）的路由器
	r := gin.Default()

	// 最简单的 GET 路由
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// 返回纯文本
	r.GET("/hello", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})

	// 启动服务器，默认监听 :8080
	r.Run()
}
