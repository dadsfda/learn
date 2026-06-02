# 第八章：测试与部署

> 学习 Gin 应用的单元测试、构建和部署。

## 目标

- 使用 `httptest` 编写接口测试
- 了解 Gin 应用的构建与部署

## 1. 单元测试

Gin 提供了便捷的测试方式，无需启动真实服务器：

```go
package main

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

// 设置测试路由器
func setupRouter() *gin.Engine {
    gin.SetMode(gin.TestMode)
    r := gin.Default()

    r.GET("/ping", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "pong"})
    })

    return r
}

func TestPing(t *testing.T) {
    router := setupRouter()

    // 创建测试请求
    req, _ := http.NewRequest("GET", "/ping", nil)
    w := httptest.NewRecorder()

    // 执行请求
    router.ServeHTTP(w, req)

    // 断言
    if w.Code != http.StatusOK {
        t.Errorf("期望状态码 200，实际 %d", w.Code)
    }

    var response map[string]string
    json.Unmarshal(w.Body.Bytes(), &response)

    if response["message"] != "pong" {
        t.Errorf("期望 message=pong，实际 %s", response["message"])
    }
}
```

## 2. 测试 POST 请求

```go
func TestLogin(t *testing.T) {
    router := setupRouter()

    // JSON 请求体
    body := strings.NewReader(`{"username":"admin","password":"123456"}`)
    req, _ := http.NewRequest("POST", "/login", body)
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("期望 200，实际 %d", w.Code)
    }
}
```

## 3. 测试表单数据

```go
func TestForm(t *testing.T) {
    router := setupRouter()

    // 表单数据
    form := url.Values{}
    form.Set("username", "admin")
    form.Set("password", "123456")

    req, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("期望 200，实际 %d", w.Code)
    }
}
```

## 4. 测试中间件

```go
func TestAuthMiddleware(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.Default()

    r.Use(AuthRequired())
    r.GET("/protected", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "ok"})
    })

    // 测试无 token
    req, _ := http.NewRequest("GET", "/protected", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusUnauthorized {
        t.Errorf("无 token 应返回 401，实际 %d", w.Code)
    }

    // 测试有 token
    req, _ = http.NewRequest("GET", "/protected", nil)
    req.Header.Set("Authorization", "Bearer my-token")
    w = httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("有 token 应返回 200，实际 %d", w.Code)
    }
}
```

## 5. 运行测试

```bash
# 运行所有测试
go test ./...

# 运行当前目录测试
go test .

# 详细输出
go test -v .

# 运行指定测试函数
go test -v -run TestPing .

# 生成覆盖率报告
go test -cover .
```

## 6. 构建与部署

### 编译

```bash
# 编译当前平台
go build -o myapp .

# 交叉编译（Linux）
GOOS=linux GOARCH=amd64 go build -o myapp .

# 交叉编译（Windows）
GOOS=windows GOARCH=amd64 go build -o myapp.exe .
```

### Docker 部署

创建 `Dockerfile`：

```dockerfile
# 多阶段构建
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o myapp .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/myapp .
EXPOSE 8080
CMD ["./myapp"]
```

构建并运行：

```bash
docker build -t myapp .
docker run -p 8080:8080 myapp
```

### Docker Compose

```yaml
version: '3.8'
services:
  web:
    build: .
    ports:
      - "8080:8080"
    environment:
      - GIN_MODE=release
    restart: unless-stopped
```

## 7. 生产环境最佳实践

```go
func main() {
    // 设置为 release 模式（减少日志输出）
    gin.SetMode(gin.ReleaseMode)

    r := gin.New()

    // 使用中间件
    r.Use(gin.Recovery())
    r.Use(CORS())
    r.Use(RequestLogger())

    // 健康检查
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })

    // 优雅关停
    srv := &http.Server{
        Addr:    ":8080",
        Handler: r,
    }

    go func() {
        if err := srv.ListenAndServe(); err != nil {
            log.Fatal(err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
}
```

## 练习

1. 为第二章的 CRUD API 编写完整的单元测试
2. 使用 Docker 部署你的 Gin 应用
3. 添加健康检查端点和优雅关停

## 恭喜！

你已经完成了 Gin 框架的基础学习！接下来可以：

- 阅读 [Gin 官方文档](https://gin-gonic.com/zh-cn/docs/) 深入了解更多功能
- 学习 [gin-contrib](https://github.com/gin-contrib) 中间件库
- 结合 GORM、Redis 等构建完整的后端应用
