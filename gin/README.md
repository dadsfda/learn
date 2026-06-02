# Gin Web Framework 学习教程

> 基于 [Gin 官方文档](https://gin-gonic.com/zh-cn/docs/) 制作，适合 Go Web 开发零基础学习者。

## 为什么选择 Gin？

- **高性能**：基于 httprouter，比 Martini 快 40 倍
- **零分配路由**：极高的内存效率
- **丰富的中间件生态**：认证、日志、CORS、限流等开箱即用
- **不会崩溃**：内置 recovery 中间件防止 panic 导致服务挂掉
- **社区活跃**：GitHub 80k+ Star，大量生产环境验证

## 前置条件

- Go 1.21 或更高版本（推荐 1.25+）
- 了解 Go 基础语法（变量、函数、结构体、接口）
- 任意代码编辑器（推荐 VS Code + Go 扩展）

## 教程目录

| 章节 | 内容 | 关键知识点 |
|------|------|-----------|
| [01 快速入门](chapter01-quickstart/) | 搭建环境，运行第一个 Gin 应用 | `gin.Default()`, `c.JSON()`, 路由注册 |
| [02 路由基础](chapter02-routing/) | HTTP 方法、路径参数、路由分组 | GET/POST/PUT/DELETE、路由分组、重定向 |
| [03 数据绑定与验证](chapter03-binding/) | 请求数据绑定、结构体验证 | ShouldBind、自定义验证器、binding tag |
| [04 响应渲染](chapter04-rendering/) | JSON/XML/HTML 等多种响应格式 | c.JSON、c.HTML、模板渲染、静态文件 |
| [05 中间件](chapter05-middleware/) | 中间件原理与自定义中间件 | 中间件链、认证、CORS、依赖注入 |
| [06 日志](chapter06-logging/) | 日志配置与自定义格式 | 日志文件输出、自定义格式、结构化日志 |
| [07 服务器配置](chapter07-config/) | HTTP 配置、优雅关停、HTTPS | 自定义端口、Graceful Shutdown、HTTP/2 |
| [08 测试与部署](chapter08-testing/) | 单元测试、构建与部署 | httptest、Docker 部署、生产最佳实践 |

## 快速开始

```bash
# 进入第一章
cd chapter01-quickstart

# 初始化模块
go mod init gin-learn

# 安装 Gin
go get -u github.com/gin-gonic/gin

# 运行
go run main.go

# 浏览器访问 http://localhost:8080/ping
```

## 学习建议

1. **按顺序学习**：每章都建立在前一章的基础上
2. **动手实践**：每个示例都要自己敲一遍并运行
3. **修改实验**：运行成功后尝试修改代码，观察效果
4. **看错误信息**：故意制造错误，学会阅读 Go 和 Gin 的错误提示
