package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 模拟数据
var users = []map[string]interface{}{
	{"id": "1", "name": "张三", "age": 25},
	{"id": "2", "name": "李四", "age": 30},
}

func main() {
	r := gin.Default()

	// ===== 1. 基础路由 =====
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// ===== 2. 路径参数 =====
	r.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		for _, u := range users {
			if u["id"] == id {
				c.JSON(200, u)
				return
			}
		}
		c.JSON(404, gin.H{"error": "用户不存在"})
	})

	// ===== 3. 查询参数 =====
	r.GET("/search", func(c *gin.Context) {
		q := c.DefaultQuery("q", "")
		page := c.DefaultQuery("page", "1")
		size := c.DefaultQuery("size", "10")

		c.JSON(200, gin.H{
			"query": q,
			"page":  page,
			"size":  size,
		})
	})

	// ===== 4. POST 表单 =====
	r.POST("/login", func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")
		c.JSON(200, gin.H{
			"username": username,
			"logged_in": password != "",
		})
	})

	// ===== 5. 路由分组 =====
	api := r.Group("/api/v1")
	{
		// 用户组
		users := api.Group("/users")
		{
			users.GET("/", func(c *gin.Context) {
				c.JSON(200, users)
			})
			users.POST("/", func(c *gin.Context) {
				name := c.PostForm("name")
				c.JSON(201, gin.H{"message": "用户已创建", "name": name})
			})
		}

		// 文章组
		articles := api.Group("/articles")
		{
			articles.GET("/", func(c *gin.Context) {
				c.JSON(200, gin.H{"articles": []string{"文章1", "文章2"}})
			})
		}
	}

	// ===== 6. 重定向 =====
	r.GET("/old", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/new")
	})
	r.GET("/new", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "这是新路径"})
	})

	// ===== 7. 通配符参数 =====
	r.GET("/files/*filepath", func(c *gin.Context) {
		filepath := c.Param("filepath")
		c.JSON(200, gin.H{"filepath": filepath})
	})

	fmt.Println("服务器启动在 http://localhost:8080")
	r.Run()
}
