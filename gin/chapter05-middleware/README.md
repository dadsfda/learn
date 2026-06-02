# 第五章：中间件

> 学习 Gin 中间件的原理、使用方法和自定义中间件编写。

## 目标

- 理解中间件的工作原理
- 使用内置中间件
- 编写自定义中间件
- 掌握中间件的执行顺序

## 1. 什么是中间件？

中间件是在请求到达处理函数**之前**和响应返回**之后**执行的代码。它像一个"洋葱"模型：

```
请求 → 中间件A(前) → 中间件B(前) → Handler → 中间件B(后) → 中间件A(后) → 响应
```

典型用途：
- **日志记录**：记录每个请求的耗时
- **认证授权**：检查用户是否登录
- **CORS**：处理跨域请求
- **限流**：防止接口被滥用
- **错误恢复**：捕获 panic 防止服务崩溃

## 2. 使用内置中间件

```go
// 使用默认中间件（Logger + Recovery）
r := gin.Default()

// 等价于
r := gin.New()
r.Use(gin.Logger())
r.Use(gin.Recovery())
```

### Recovery 中间件

防止 panic 导致整个服务器崩溃：

```go
r := gin.New()
r.Use(gin.Recovery())

r.GET("/crash", func(c *gin.Context) {
    panic("something went wrong!") // 不会崩溃，Recovery 会捕获
})
```

## 3. 自定义中间件

中间件是一个返回 `gin.HandlerFunc` 的函数：

```go
func MyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // ===== 请求前逻辑 =====
        start := time.Now()

        // c.Next() 将控制权交给下一个中间件/处理函数
        // 等待后续处理完成后，继续执行下面的代码
        c.Next()

        // ===== 响应后逻辑 =====
        duration := time.Since(start)
        fmt.Printf("请求耗时: %v\n", duration)
    }
}
```

## 4. 中间件执行顺序

```go
func main() {
    r := gin.Default()

    // 全局中间件 - 对所有路由生效
    r.Use(Logger())

    r.GET("/test", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "ok"})
    })
}

func Logger() gin.HandlerFunc {
    return func(c *gin.Context) {
        fmt.Println("1. Logger 开始")
        c.Next()
        fmt.Println("4. Logger 结束")
    }
}

// 输出顺序：
// 1. Logger 开始
// 2. Handler 开始
// 3. Handler 结束
// 4. Logger 结束
```

### 使用 `c.Abort()` 中断

```go
func AuthRequired() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            // Abort() 停止执行后续中间件和 Handler
            c.AbortWithStatusJSON(401, gin.H{"error": "未授权"})
            return
        }
        // 验证通过，继续执行
        c.Next()
    }
}
```

## 5. 中间件作用域

```go
func main() {
    r := gin.New()

    // 1. 全局中间件
    r.Use(gin.Logger())

    // 2. 路由组中间件
    admin := r.Group("/admin", AuthRequired())
    {
        admin.GET("/dashboard", dashboard)
    }

    // 3. 单个路由中间件
    r.GET("/profile", AuthRequired(), getProfile)
}
```

## 6. 实用中间件示例

### 请求日志中间件

```go
func RequestLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        method := c.Request.Method

        c.Next()

        latency := time.Since(start)
        status := c.Writer.Status()

        log.Printf("[%s] %s %s %d %v",
            method, path, c.ClientIP(), status, latency)
    }
}
```

### CORS 中间件

```go
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
```

### 限流中间件

```go
import "golang.org/x/time/rate"

func RateLimit(rps float64, burst int) gin.HandlerFunc {
    limiter := rate.NewLimiter(rate.Limit(rps), burst)

    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.AbortWithStatusJSON(429, gin.H{"error": "请求过于频繁"})
            return
        }
        c.Next()
    }
}
```

## 7. 中间件中的 Goroutine

在中间件中启动 goroutine 时，**不能直接使用原始的 `*gin.Context`**：

```go
// ❌ 错误：goroutine 可能在请求结束后才执行
func BadMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        go func() {
            // c 可能已经被回收！
            fmt.Println(c.Request.URL.Path)
        }()
    }
}

// ✅ 正确：先复制需要的数据
func GoodMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        path := c.Request.URL.Path // 先复制
        go func() {
            fmt.Println(path) // 使用复制的数据
        }()
    }
}
```

## 练习

1. 编写一个中间件，记录每个请求的处理耗时
2. 实现一个简单的认证中间件，检查请求头中的 token
3. 编写一个 IP 白名单中间件，只允许指定 IP 访问

## 下一章

[第六章：日志](../chapter06-logging/) — 学习日志配置与自定义格式。
