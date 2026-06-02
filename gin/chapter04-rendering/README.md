# 第四章：响应渲染

> 学习 Gin 的多种响应格式：JSON、XML、HTML 模板、静态文件等。

## 目标

- 掌握 JSON、XML、YAML 等格式的响应
- 使用 HTML 模板渲染页面
- 提供静态文件服务

## 1. JSON 响应

```go
// 最常用的方式
c.JSON(http.StatusOK, gin.H{
    "name": "张三",
    "age":  25,
})

// 使用结构体（推荐，更规范）
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

c.JSON(http.StatusOK, User{Name: "张三", Age: 25})
```

### JSON 相关变体

```go
// SecureJSON - 防止 JSON 劫持（在 JSON 前加 )]} 前缀）
c.SecureJSON(http.StatusOK, gin.H{"data": "sensitive"})

// JSONP - 跨域 JSON
c.JSONP(http.StatusOK, gin.H{"data": "cross-origin"})

// PureJSON - 不转义 HTML 特殊字符
c.PureJSON(http.StatusOK, gin.H{"html": "<b>bold</b>"})

// AsciiJSON - ASCII 编码
c.AsciiJSON(http.StatusOK, gin.H{"name": "张三"})
// 输出: {"name":"张三"}
```

## 2. XML 响应

```go
c.XML(http.StatusOK, gin.H{
    "name": "张三",
    "age":  25,
})
// 输出:
// <map><name>张三</name><age>25</age></map>
```

## 3. YAML 响应

```go
c.YAML(http.StatusOK, gin.H{
    "name": "张三",
    "age":  25,
})
```

## 4. HTML 模板渲染

### 基础模板

创建 `templates/index.html`：

```html
<!DOCTYPE html>
<html>
<head><title>{{ .title }}</title></head>
<body>
    <h1>Hello, {{ .name }}!</h1>
</body>
</html>
```

### Go 代码

```go
// 加载模板
r.LoadHTMLGlob("templates/*")

r.GET("/", func(c *gin.Context) {
    c.HTML(http.StatusOK, "index.html", gin.H{
        "title": "首页",
        "name":  "Gin",
    })
})
```

### 多模板目录

```go
// 加载多个目录的模板
r.LoadHTMLGlob("templates/**/*")

// 或使用 embed（推荐，编译到二进制文件中）
//go:embed templates/*
var templateFS embed.FS

tmpl := template.Must(template.New("").ParseFS(templateFS, "templates/*"))
r.SetHTMLTemplate(tmpl)
```

## 5. 静态文件服务

```go
// 将 /static 路径映射到 ./assets 目录
r.Static("/static", "./assets")

// 单个静态文件
r.StaticFile("/favicon.ico", "./assets/favicon.ico")

// 提供文件下载
r.GET("/download", func(c *gin.Context) {
    c.File("./files/report.pdf")
})

// 从 io.Reader 提供数据
r.GET("/data", func(c *gin.Context) {
    c.DataFromReader(http.StatusOK, contentLength, contentType, reader, extraHeaders)
})
```

## 6. 重定向

```go
// 301 永久重定向
r.GET("/old", func(c *gin.Context) {
    c.Redirect(http.StatusMovedPermanently, "/new")
})

// 302 临时重定向
r.GET("/temp", func(c *gin.Context) {
    c.Redirect(http.StatusFound, "/other")
})
```

## 7. 响应头和 Cookie

```go
r.GET("/headers", func(c *gin.Context) {
    // 设置响应头
    c.Header("X-Custom-Header", "value")
    c.Header("X-Request-Id", "abc123")

    // 设置 Cookie
    c.SetCookie("session_id", "xyz", 3600, "/", "localhost", false, true)

    c.JSON(200, gin.H{"message": "check headers"})
})
```

## 练习

1. 创建一个接口同时支持 JSON 和 XML 响应（根据 Accept 头自动选择）
2. 创建一个简单的 HTML 页面，显示用户的个人信息
3. 实现一个文件下载接口

## 下一章

[第五章：中间件](../chapter05-middleware/) — 学习中间件的原理与自定义。
