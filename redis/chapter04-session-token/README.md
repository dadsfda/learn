# 第四章：登录会话与验证码

> Redis 很适合保存短期有效的登录状态、验证码和一次性操作状态。

## 目标

- 用 Redis 保存登录 token
- 用 Redis 保存短信验证码
- 理解临时状态为什么必须设置过期时间

## 1. 保存登录 token

用户登录成功后，服务端生成 token，并写入 Redis：

```bash
SET login:token:abc123 user:1001 EX 7200
GET login:token:abc123
TTL login:token:abc123
```

含义：

- `login:token:abc123` 是登录凭证
- value 保存用户身份，例如 `user:1001`
- `EX 7200` 表示 2 小时后自动过期

用户退出登录时主动删除：

```bash
DEL login:token:abc123
```

## 2. 保存短信验证码

验证码天然是短期数据：

```bash
SET verify:phone:13800000000 "246810" EX 300
GET verify:phone:13800000000
DEL verify:phone:13800000000
```

校验流程：

1. 用户提交手机号和验证码。
2. 服务端从 Redis 读取验证码。
3. 不存在：验证码过期或未发送。
4. 不相等：验证码错误。
5. 相等：校验通过，并删除验证码。

校验通过后删除验证码，是为了防止同一个验证码重复使用。

## 3. 防止验证码频繁发送

发送验证码前，先检查冷却 key：

```bash
SET verify:cooldown:13800000000 1 NX EX 60
```

含义：

- `NX` 表示 key 不存在时才写入
- 写入成功：允许发送验证码
- 写入失败：说明 60 秒内已经发送过

发送成功后再写验证码：

```bash
SET verify:phone:13800000000 "246810" EX 300
```

## 4. 一次性操作状态

例如重置密码链接：

```bash
SET reset-password:ticket:tk_123 user:1001 EX 900
GET reset-password:ticket:tk_123
DEL reset-password:ticket:tk_123
```

用户点开链接并完成重置后，要删除 ticket，避免重复使用。

## 5. Go 伪代码：校验验证码

```go
func CheckCode(phone string, input string) bool {
    key := fmt.Sprintf("verify:phone:%s", phone)

    code, err := redis.Get(ctx, key).Result()
    if err == redis.Nil {
        return false
    }
    if err != nil {
        return false
    }
    if code != input {
        return false
    }

    redis.Del(ctx, key)
    return true
}
```

## 实际应用：登录态设计

简单后台系统可以这样做：

```text
用户登录
-> 生成随机 token
-> Redis 写入 login:token:{token} = user_id，过期 2 小时
-> 前端后续请求携带 token
-> 后端查 Redis，拿到 user_id 就认为已登录
```

这种方案的优点：

- 服务端可以主动让 token 失效
- 过期时间由 Redis 自动管理
- 多个后端实例可以共享登录状态

## 练习

1. 写入一个 30 分钟过期的登录 token。
2. 写入一个 5 分钟过期的验证码。
3. 使用 `SET key value NX EX 60` 模拟发送验证码冷却。
4. 校验成功后删除验证码 key。

## 下一章

[第五章：计数器与限流](../chapter05-counter-rate-limit/) — 学习高频计数和接口限流。

