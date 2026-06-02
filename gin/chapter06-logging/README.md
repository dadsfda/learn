# 第六章：日志

> 学习 Gin 的日志系统：配置、自定义格式、文件输出。

## 目标

- 配置日志输出到文件
- 自定义日志格式
- 控制日志颜色和输出

## 1. 默认日志

`gin.Default()` 自带 Logger 中间件，会输出彩色日志到控制台：

```
[GIN] 2026/06/02 - 15:00:00 | 200 | 125µs | 127.0.0.1 | GET "/ping"
```

## 2. 日志输出到文件

```go
import "os"

func main() {
    // 创建日志文件
    f, _ := os.Create("gin.log")
    gin.DefaultWriter = io.MultiWriter(f, os.Stdout) // 同时写入文件和控制台

    r := gin.Default()
    r.Run()
}
```

## 3. 自定义日志格式

```go
func main() {
    r := gin.New()

    // 自定义日志格式
    r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
            param.ClientIP,
            param.TimeStamp.Format(time.RFC1123),
            param.Method,
            param.Path,
            param.Request.Proto,
            param.StatusCode,
            param.Latency,
            param.Request.UserAgent(),
            param.ErrorMessage,
        )
    }))

    r.Use(gin.Recovery())
    r.Run()
}
```

## 4. 跳过特定路径的日志

```go
r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
    SkipPaths: []string{"/health", "/metrics"}, // 不记录健康检查和指标接口
}))
```

## 5. 控制日志颜色

```go
// 禁用颜色（生产环境推荐）
gin.DisableConsoleColor()

// 强制启用颜色
gin.ForceConsoleColor()
```

## 6. 结构化日志（使用 slog）

Go 1.21+ 内置 `log/slog` 包，推荐在生产环境使用：

```go
import "log/slog"

func SlogMiddleware() gin.HandlerFunc {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    return func(c *gin.Context) {
        start := time.Now()
        c.Next()

        logger.Info("request",
            "method", c.Request.Method,
            "path", c.Request.URL.Path,
            "status", c.Writer.Status(),
            "latency", time.Since(start).String(),
            "client_ip", c.ClientIP(),
        )
    }
}
```

## 练习

1. 将日志同时输出到文件和控制台
2. 自定义日志格式，包含请求 ID
3. 使用 slog 实现结构化 JSON 日志

## 下一章

[第七章：服务器配置](../chapter07-config/) — 学习自定义 HTTP 配置和优雅关停。
