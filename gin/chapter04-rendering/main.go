package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 用户结构体（带 JSON tag 控制输出字段名）
type User struct {
	Name  string `json:"name" xml:"name"`
	Age   int    `json:"age" xml:"age"`
	Email string `json:"email" xml:"email"`
}

func main() {
	r := gin.Default()

	// 1. JSON 响应
	r.GET("/json", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "这是 JSON 响应",
			"data":    gin.H{"name": "张三", "age": 25},
		})
	})

	// 2. 结构体 JSON 响应
	r.GET("/user", func(c *gin.Context) {
		user := User{Name: "张三", Age: 25, Email: "zhangsan@example.com"}
		c.JSON(http.StatusOK, user)
	})

	// 3. XML 响应
	r.GET("/xml", func(c *gin.Context) {
		user := User{Name: "张三", Age: 25, Email: "zhangsan@example.com"}
		c.XML(http.StatusOK, user)
	})

	// 4. YAML 响应
	r.GET("/yaml", func(c *gin.Context) {
		user := User{Name: "张三", Age: 25, Email: "zhangsan@example.com"}
		c.YAML(http.StatusOK, user)
	})

	// 5. 纯文本响应
	r.GET("/text", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, %s! 你今年 %d 岁", "张三", 25)
	})

	// 6. HTML 模板渲染
	r.LoadHTMLGlob("templates/*")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Gin 学习",
			"name":  "张四",
		})
	})

	// 7. 静态文件服务
	r.Static("/static", "./static")
	r.StaticFile("/favicon.ico", "./static/favicon.ico")

	// 8. 文件下载
	r.GET("/download", func(c *gin.Context) {
		c.Header("Content-Disposition", "attachment; filename=readme.txt")
		c.File("./static/readme.txt")
	})

	// 9. 重定向
	r.GET("/old-path", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/new-path")
	})
	r.GET("/new-path", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "这是新路径"})
	})

	// 10. 设置响应头和 Cookie
	r.GET("/headers", func(c *gin.Context) {
		c.Header("X-Custom-Header", "my-value")
		c.SetCookie("session_id", "abc123", 3600, "/", "localhost", false, true)
		c.JSON(200, gin.H{"message": "查看响应头和 Cookie"})
	})

	r.Run()
}
