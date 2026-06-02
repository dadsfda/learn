# 第二章：路由基础

> 学习 Gin 的路由系统：HTTP 方法、路径参数、查询参数、路由分组。

## 目标

- 掌握所有 HTTP 方法的路由注册
- 使用路径参数和查询参数
- 理解路由分组的作用
- 处理表单和文件上传

## 1. HTTP 方法

Gin 为每种 HTTP 方法提供了对应的注册函数：

```go
r.GET("/users", listUsers)       // 查询列表
r.POST("/users", createUser)     // 创建
r.PUT("/users/:id", updateUser)  // 更新（全量）
r.DELETE("/users/:id", deleteUser) // 删除
r.PATCH("/users/:id", patchUser)   // 更新（部分）

// 匹配所有 HTTP 方法
r.Any("/health", healthCheck)
```

## 2. 路径参数

用 `:name` 定义路径参数，通过 `c.Param("name")` 获取：

```go
// 路由定义
r.GET("/users/:id", func(c *gin.Context) {
    id := c.Param("id") // 获取路径参数
    c.JSON(200, gin.H{"user_id": id})
})

// 访问 GET /users/123
// 返回 {"user_id": "123"}
```

### 通配符参数

用 `*name` 匹配剩余所有路径：

```go
r.GET("/files/*filepath", func(c *gin.Context) {
    filepath := c.Param("filepath")
    c.JSON(200, gin.H{"filepath": filepath})
})

// 访问 GET /files/documents/readme.md
// 返回 {"filepath": "/documents/readme.md"}
```

## 3. 查询字符串参数

用 `c.Query()` 获取 URL 中 `?` 后面的参数：

```go
// GET /search?q=gin&page=1
r.GET("/search", func(c *gin.Context) {
    q := c.DefaultQuery("q", "default")  // 有默认值
    page := c.Query("page")              // 无默认值，空字符串

    c.JSON(200, gin.H{
        "query": q,
        "page":  page,
    })
})
```

| 方法 | 说明 |
|------|------|
| `c.Query("key")` | 获取参数，不存在返回 `""` |
| `c.DefaultQuery("key", "default")` | 获取参数，不存在返回默认值 |
| `c.GetQuery("key")` | 获取参数和是否存在（bool） |

## 4. POST 表单数据

```go
r.POST("/login", func(c *gin.Context) {
    username := c.PostForm("username")
    password := c.DefaultPostForm("password", "")

    c.JSON(200, gin.H{
        "username": username,
    })
})

// curl 测试：
// curl -X POST http://localhost:8080/login -d "username=admin&password=123"
```

## 5. 路由分组

将相关路由组织在一起，共享前缀和中间件：

```go
// 用户相关路由
userGroup := r.Group("/api/v1/users")
{
    userGroup.GET("/", listUsers)         // GET /api/v1/users/
    userGroup.POST("/", createUser)       // POST /api/v1/users/
    userGroup.GET("/:id", getUser)        // GET /api/v1/users/:id
    userGroup.PUT("/:id", updateUser)     // PUT /api/v1/users/:id
    userGroup.DELETE("/:id", deleteUser)  // DELETE /api/v1/users/:id
}

// 文章相关路由
articleGroup := r.Group("/api/v1/articles")
{
    articleGroup.GET("/", listArticles)
    articleGroup.POST("/", createArticle)
}
```

### 分组 + 中间件

```go
// 所有 admin 路由都需要认证
admin := r.Group("/admin", AuthRequired())
{
    admin.GET("/dashboard", dashboard)
    admin.GET("/settings", settings)
}
```

## 6. 重定向

```go
// HTTP 重定向（301）
r.GET("/old-path", func(c *gin.Context) {
    c.Redirect(http.StatusMovedPermanently, "/new-path")
})

// 路由重定向
r.GET("/test", func(c *gin.Context) {
    c.Request.URL.Path = "/test2"
    r.HandleContext(c)
})
```

## 7. 不同 Content-Type 的请求处理

```go
// JSON 请求体
r.POST("/json", func(c *gin.Context) {
    var json struct {
        Name string `json:"name"`
    }
    if err := c.ShouldBindJSON(&json); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"name": json.Name})
})

// 表单请求
r.POST("/form", func(c *gin.Context) {
    name := c.PostForm("name")
    c.JSON(200, gin.H{"name": name})
})

// 查询参数 + 表单混合
r.POST("/mixed", func(c *gin.Context) {
    id := c.Query("id")        // 从 URL 获取
    name := c.PostForm("name") // 从 body 获取
    c.JSON(200, gin.H{"id": id, "name": name})
})
```

## 练习

1. 创建一个简单的 REST API，实现用户的 CRUD 操作（增删改查）
2. 使用路由分组将 `/api/v1/users` 和 `/api/v1/posts` 分开
3. 实现一个搜索接口，支持分页参数 `page` 和 `size`

## 下一章

[第三章：数据绑定与验证](../chapter03-binding/) — 学习请求数据的自动绑定和验证。
