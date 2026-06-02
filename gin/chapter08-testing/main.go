package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRouter 配置路由（方便测试复用）
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 简单的 GET
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// 登录接口（支持 JSON 和表单）
	r.POST("/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" form:"username" binding:"required"`
			Password string `json:"password" form:"password" binding:"required"`
		}

		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Username == "admin" && req.Password == "123456" {
			c.JSON(http.StatusOK, gin.H{
				"token":   "mock-jwt-token",
				"message": "登录成功",
			})
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
	})

	// 用户 API
	r.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{
			"id":   id,
			"name": "张三",
		})
	})

	return r
}

func main() {
	r := SetupRouter()
	r.Run()
}
