# 第五章：计数器与限流

> Redis 的自增命令是原子的，非常适合做计数器和简单限流。

## 目标

- 使用 `INCR` 实现计数器
- 使用 `INCR + EXPIRE` 实现固定窗口限流
- 理解原子操作的价值

## 1. 阅读量计数

文章每被访问一次，阅读量加 1：

```bash
INCR article:1001:view_count
INCR article:1001:view_count
GET article:1001:view_count
```

`INCR` 是原子操作，多个请求同时执行也不会互相覆盖。

## 2. 点赞数计数

如果只需要显示点赞总数：

```bash
INCR article:1001:like_count
DECR article:1001:like_count
GET article:1001:like_count
```

如果还要防止重复点赞，应该结合 Set：

```bash
SADD article:1001:liked_users user:1001
SCARD article:1001:liked_users
```

## 3. 每日计数

按天统计接口调用量：

```bash
INCR api:login:count:2026-06-06
EXPIRE api:login:count:2026-06-06 172800
GET api:login:count:2026-06-06
```

日期放进 key 里，查询和清理都很直观。

## 4. 固定窗口限流

需求：同一个 IP 每分钟最多访问登录接口 5 次。

```bash
INCR rate:login:127.0.0.1:202606061530
EXPIRE rate:login:127.0.0.1:202606061530 70
GET rate:login:127.0.0.1:202606061530
```

业务判断：

- 第一次访问时 key 不存在，`INCR` 后变成 1。
- 如果计数小于等于 5，允许访问。
- 如果计数大于 5，拒绝访问。
- key 过期后进入下一个窗口。

## 5. 更简单的限流 key

如果不想把时间拼进 key，也可以让 Redis 自己过期：

```bash
INCR rate:login:127.0.0.1
EXPIRE rate:login:127.0.0.1 60
```

但要注意：每次都执行 `EXPIRE` 会让窗口被刷新，可能变成“距离最后一次请求 60 秒”。实际项目里通常只在第一次 `INCR` 后设置过期时间。

## 6. Go 伪代码：简单限流

```go
func AllowLogin(ip string) bool {
    key := fmt.Sprintf("rate:login:%s", ip)

    count, err := redis.Incr(ctx, key).Result()
    if err != nil {
        return false
    }

    if count == 1 {
        redis.Expire(ctx, key, time.Minute)
    }

    return count <= 5
}
```

生产环境中，如果必须保证 `INCR` 和首次 `EXPIRE` 严格一起成功，可以使用 Lua 脚本；小项目先理解这个流程即可。

## 实际应用：防刷接口

适合做限流的接口：

- 登录
- 发送验证码
- 搜索
- 评论发布
- 密码重置

限流维度可以是：

- IP
- 用户 ID
- 手机号
- 接口路径

示例 key：

```text
rate:sms:phone:13800000000
rate:login:ip:127.0.0.1
rate:comment:user:1001
```

## 练习

1. 用 `INCR` 统计文章阅读量。
2. 用日期 key 统计今天的登录次数。
3. 模拟同一个 IP 访问 6 次登录接口，第 6 次应被拒绝。
4. 思考：验证码发送限流应该按手机号、IP，还是两者都要？

## 下一章

[第六章：排行榜](../chapter06-rank-list/) — 学习用 ZSet 实现排名功能。

