# 第七章：服务器配置

> 学习 HTTP 服务器配置、优雅关停、Cookie 和 WebSocket 等。

## 目标

- 自定义 HTTP 服务器配置
- 实现优雅关停（Graceful Shutdown）
- 使用 Cookie 和 WebSocket

## 1. 自定义端口和地址

```go
// 默认监听 :8080
r.Run()

// 指定端口
r.Run(":9090")

// 指定地址和端口
r.Run("127.0.0.1:9090")
```

## 2. 自定义 HTTP 配置

```go
import "net/http"

func main() {
    r := gin.Default()

    // 自定义 http.Server 配置
    srv := &http.Server{
        Addr:              ":8080",
        Handler:           r,
        ReadTimeout:       10 * time.Second,
        WriteTimeout:      10 * time.Second,
        MaxHeaderBytes:    1 << 20, // 1MB
        ReadHeaderTimeout: 5 * time.Second,
    }

    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("服务器启动失败: %v", err)
    }
}
```

## 3. 优雅关停（Graceful Shutdown）

在服务重启或停止时，等待正在处理的请求完成：

```go
import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.Default()

    r.GET("/slow", func(c *gin.Context) {
        time.Sleep(5 * time.Second) // 模拟慢请求
        c.JSON(200, gin.H{"message": "done"})
    })

    srv := &http.Server{
        Addr:    ":8080",
        Handler: r,
    }

    // 启动服务器（非阻塞）
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("服务器错误: %v", err)
        }
    }()

    // 等待中断信号
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("正在关闭服务器...")

    // 给 5 秒时间处理剩余请求
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("服务器关闭失败: %v", err)
    }

    log.Println("服务器已安全关闭")
}
```

## 4. Cookie 操作

```go
// 设置 Cookie
r.GET("/set-cookie", func(c *gin.Context) {
    c.SetCookie(
        "session_id",  // 名称
        "abc123",      // 值
        3600,          // 过期时间（秒）
        "/",           // 路径
        "localhost",   // 域名
        false,         // Secure（仅 HTTPS）
        true,          // HttpOnly（禁止 JS 访问）
    )
    c.JSON(200, gin.H{"message": "Cookie 已设置"})
})

// 读取 Cookie
r.GET("/get-cookie", func(c *gin.Context) {
    sessionID, err := c.Cookie("session_id")
    if err != nil {
        c.JSON(400, gin.H{"error": "Cookie 不存在"})
        return
    }
    c.JSON(200, gin.H{"session_id": sessionID})
})

// 删除 Cookie
r.GET("/delete-cookie", func(c *gin.Context) {
    c.SetCookie("session_id", "", -1, "/", "localhost", false, true)
    c.JSON(200, gin.H{"message": "Cookie 已删除"})
})
```

## 5. WebSocket 支持

```go
import "github.com/gorilla/websocket"

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // 允许所有来源（生产环境应限制）
    },
}

r.GET("/ws", func(c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }
    defer conn.Close()

    for {
        // 读取消息
        messageType, message, err := conn.ReadMessage()
        if err != nil {
            break
        }

        // 回显消息
        if err := conn.WriteMessage(messageType, message); err != nil {
            break
        }
    }
})
```

## 6. 健康检查端点

```go
r.GET("/health", func(c *gin.Context) {
    c.JSON(200, gin.H{
        "status": "ok",
        "time":   time.Now().Format(time.RFC3339),
    })
})
```

## 练习

1. 实现一个支持优雅关停的 HTTP 服务器
2. 实现一个简单的会话管理（基于 Cookie）
3. 添加健康检查端点

## 下一章

[第八章：测试与部署](../chapter08-testing/) — 学习单元测试和部署最佳实践。
