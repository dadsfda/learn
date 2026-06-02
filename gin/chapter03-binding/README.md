# 第三章：数据绑定与验证

> 学习 Gin 的请求数据自动绑定和结构体验证功能。

## 目标

- 使用 ShouldBind 自动绑定请求数据到结构体
- 掌握 binding tag 进行数据验证
- 编写自定义验证器

## 1. 基础绑定

Gin 可以自动将请求数据（JSON、表单、查询参数）绑定到 Go 结构体：

```go
// 定义结构体，使用 binding tag 指定验证规则
type LoginRequest struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required,min=6"`
}

r.POST("/login", func(c *gin.Context) {
    var req LoginRequest

    // ShouldBindJSON 绑定 JSON 请求体
    // 如果绑定失败，返回错误而不是 panic
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{
        "username": req.Username,
        "message":  "登录成功",
    })
})
```

### curl 测试

```bash
# 正确请求
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'

# 缺少字段（会返回错误）
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin"}'
```

## 2. 不同绑定方法

| 方法 | 绑定来源 | 说明 |
|------|---------|------|
| `ShouldBindJSON` | JSON 请求体 | Content-Type: application/json |
| `ShouldBindQuery` | URL 查询参数 | ?key=value |
| `ShouldBindUri` | URI 路径参数 | /users/:id |
| `ShouldBind` | 自动检测 | 根据 Content-Type 自动选择 |
| `ShouldBindWith` | 指定来源 | 可以手动指定绑定器 |

```go
// 绑定查询参数
type SearchRequest struct {
    Query  string `form:"q" binding:"required"`
    Page   int    `form:"page" default:"1"`
    Size   int    `form:"size" default:"10"`
}

r.GET("/search", func(c *gin.Context) {
    var req SearchRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, req)
})

// 绑定 URI 参数
type UserRequest struct {
    ID uint `uri:"id" binding:"required"`
}

r.GET("/users/:id", func(c *gin.Context) {
    var req UserRequest
    if err := c.ShouldBindUri(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"user_id": req.ID})
})
```

## 3. 常用验证规则

使用 `binding` tag 指定验证规则，多个规则用逗号分隔：

```go
type CreateUserRequest struct {
    // required: 必填
    // min/max: 字符串最小/最大长度，数值最小/最大值
    Username string `json:"username" binding:"required,min=3,max=20"`

    // email: 邮箱格式验证
    Email string `json:"email" binding:"required,email"`

    // oneof: 枚举值，只能是指定值之一
    Gender string `json:"gender" binding:"required,oneof=male female other"`

    // gte/lte: 大于等于 / 小于等于
    Age int `json:"age" binding:"required,gte=1,lte=150"`

    // len: 精确长度
    Phone string `json:"phone" binding:"required,len=11"`

    // contains: 必须包含指定字符串
    Bio string `json:"bio" binding:"contains=hello"`

    // 不验证，可选字段
    Nickname string `json:"nickname"`
}
```

### 验证规则速查表

| 规则 | 说明 | 示例 |
|------|------|------|
| `required` | 必填 | `binding:"required"` |
| `min` | 最小值/长度 | `binding:"min=3"` |
| `max` | 最大值/长度 | `binding:"max=100"` |
| `gte` | 大于等于 | `binding:"gte=0"` |
| `lte` | 小于等于 | `binding:"lte=150"` |
| `email` | 邮箱格式 | `binding:"email"` |
| `url` | URL 格式 | `binding:"url"` |
| `oneof` | 枚举值 | `binding:"oneof=a b c"` |
| `len` | 精确长度 | `binding:"len=11"` |
| `contains` | 包含子串 | `binding:"contains=abc"` |
| `startswith` | 以...开头 | `binding:"startswith=prefix"` |
| `endswith` | 以...结尾 | `binding:"endswith=.com"` |

## 4. 自定义验证器

当内置验证规则不够用时，可以注册自定义验证器：

```go
import "github.com/go-playground/validator/v10"

// 自定义验证函数：检查日期格式
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

    r.POST("/event", func(c *gin.Context) {
        var req struct {
            Name string `json:"name" binding:"required"`
            Date string `json:"date" binding:"required,date"` // 使用自定义验证器
        }
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, req)
    })

    r.Run()
}
```

## 5. 处理验证错误

```go
r.POST("/create", func(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        // 判断是否为验证错误
        errs, ok := err.(validator.ValidationErrors)
        if !ok {
            c.JSON(400, gin.H{"error": "参数错误"})
            return
        }

        // 返回友好的错误信息
        messages := make(map[string]string)
        for _, e := range errs {
            messages[e.Field()] = fmt.Sprintf("字段 %s 验证失败: %s", e.Field(), e.Tag())
        }

        c.JSON(400, gin.H{"errors": messages})
        return
    }

    c.JSON(200, gin.H{"message": "创建成功"})
})
```

## 练习

1. 创建一个注册接口，验证用户名（3-20位）、邮箱、密码（至少8位，包含数字和字母）
2. 为年龄字段添加自定义验证器，要求必须大于当前年份减去 100
3. 处理验证错误，返回中文友好的错误提示

## 下一章

[第四章：响应渲染](../chapter04-rendering/) — 学习 JSON、HTML、XML 等多种响应格式。
