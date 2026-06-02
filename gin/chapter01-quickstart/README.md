# 第一章：快速入门

> 搭建开发环境，运行你的第一个 Gin Web 应用。

## 目标

- 安装 Gin 框架
- 创建一个简单的 HTTP 服务器
- 理解 Gin 的基本运行流程

## 1. 初始化项目

```bash
# 创建项目目录
mkdir my-gin-app && cd my-gin-app

# 初始化 Go 模块
go mod init my-gin-app

# 安装 Gin
go get -u github.com/gin-gonic/gin
```

## 2. 最小 Gin 应用

创建 `main.go`：

```go
package main

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

func main() {
    // gin.Default() 创建一个带有 Logger 和 Recovery 中间件的路由器
    // Logger: 所有请求都会打印到控制台
    // Recovery: 捕获 panic，防止服务器崩溃
    r := gin.Default()

    // 注册一个 GET 请求的路由
    // 当用户访问 GET /ping 时，执行后面的处理函数
    r.GET("/ping", func(c *gin.Context) {
        // c.JSON() 返回 JSON 格式的响应
        // http.StatusOK = 200 状态码
        // gin.H 是 map[string]interface{} 的简写
        c.JSON(http.StatusOK, gin.H{
            "message": "pong",
        })
    })

    // 在 0.0.0.0:8080 启动服务器
    // 也可以指定端口：r.Run(":9090")
    r.Run()
}
```

## 3. 运行

```bash
go run main.go
```

输出类似：
```
[GIN-debug] Listening and serving HTTP on :8080
```

## 4. 测试

浏览器访问 `http://localhost:8080/ping`，会看到：

```json
{"message": "pong"}
```

控制台会打印请求日志：
```
[GIN] 2026/06/02 - 15:00:00 | 200 | 0s | 127.0.0.1 | GET "/ping"
```

## 核心概念解析

### `gin.Default()` vs `gin.New()`

| 方法 | 说明 |
|------|------|
| `gin.Default()` | 自带 Logger + Recovery 中间件，适合快速开发 |
| `gin.New()` | 空白路由器，不带任何中间件，需要手动添加 |

```go
// 等价写法
r := gin.New()
r.Use(gin.Logger())
r.Use(gin.Recovery())
```

### `gin.H` 是什么？

`gin.H` 只是 `map[string]interface{}` 的类型别名，让你写起来更简洁：

```go
// 使用 gin.H（推荐）
gin.H{"message": "pong", "code": 200}

// 等价的完整写法
map[string]interface{}{"message": "pong", "code": 200}
```

### `gin.Context` 是什么？

`gin.Context`（通常缩写为 `c`）是 Gin 最核心的结构体，它包含了一次 HTTP 请求的所有信息：

- 请求数据：URL 参数、表单数据、请求头、请求体
- 响应方法：JSON、HTML、XML、重定向等
- 中间件控制：Next()、Abort()
- 错误处理：Error()

## 练习

1. 修改端口为 `9090`，重新运行
2. 添加一个 `GET /hello` 路由，返回 `{"hello": "world"}`
3. 将 `gin.Default()` 改为 `gin.New()`，观察日志输出的变化

## 下一章

[第二章：路由基础](../chapter02-routing/) — 学习 HTTP 方法、路径参数、路由分组等。
