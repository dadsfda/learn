package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ===== 请求结构体 =====

// 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Password string `json:"password" binding:"required,min=6"`
}

// 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Email    string `json:"email" binding:"required,email"`
	Age      int    `json:"age" binding:"required,gte=1,lte=150"`
	Gender   string `json:"gender" binding:"required,oneof=male female other"`
}

// 搜索请求（从查询参数绑定）
type SearchRequest struct {
	Query string `form:"q" binding:"required"`
	Page  int    `form:"page,default=1"`
	Size  int    `form:"size,default=10"`
}

// URI 参数绑定
type UserIDRequest struct {
	ID uint `uri:"id" binding:"required"`
}

// 自定义验证：日期格式
func validateDate(fl validator.FieldLevel) bool {
	date := fl.Field().String()
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

func main() {
	r := gin.Default()

	// 注册自定义验证器
	if v, ok := r.Validator.(*validator.Validate); ok {
		v.RegisterValidation("date", validateDate)
	}

	// 1. JSON 绑定
	r.POST("/login", func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"username": req.Username,
			"message":  "登录成功",
		})
	})

	// 2. 复杂验证
	r.POST("/users", func(c *gin.Context) {
		var req CreateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			// 处理验证错误
			errs, ok := err.(validator.ValidationErrors)
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "参数格式错误"})
				return
			}
			messages := make(map[string]string)
			for _, e := range errs {
				messages[e.Field()] = e.Field() + " 验证失败: " + e.Tag()
			}
			c.JSON(http.StatusBadRequest, gin.H{"errors": messages})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "用户创建成功",
			"user":    req,
		})
	})

	// 3. 查询参数绑定
	r.GET("/search", func(c *gin.Context) {
		var req SearchRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, req)
	})

	// 4. URI 参数绑定
	r.GET("/users/:id", func(c *gin.Context) {
		var req UserIDRequest
		if err := c.ShouldBindUri(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": req.ID})
	})

	// 5. 自定义验证器
	r.POST("/event", func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required"`
			Date string `json:"date" binding:"required,date"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"event": req.Name,
			"date":  req.Date,
		})
	})

	r.Run()
}
