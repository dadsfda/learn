# 第一章：快速入门

> 启动 Redis，掌握最基础的 key/value 操作和过期时间。

## 目标

- 启动一个本地 Redis
- 使用 `redis-cli` 读写数据
- 理解 key、value、过期时间的基本概念

## 1. 启动 Redis

使用 Docker：

```bash
docker run --name redis-learn -p 6379:6379 -d redis:7
docker exec -it redis-learn redis-cli
```

测试连接：

```bash
PING
```

输出：

```text
PONG
```

## 2. 写入和读取字符串

Redis 最基础的模型是 key/value：

```bash
SET user:1:name "小明"
GET user:1:name
```

输出：

```text
"小明"
```

删除 key：

```bash
DEL user:1:name
GET user:1:name
```

删除后会返回空值：

```text
(nil)
```

## 3. 设置过期时间

很多 Redis 数据都是临时数据，例如验证码、登录 token、缓存。临时数据应该设置过期时间：

```bash
SET verify:phone:13800000000 "246810"
EXPIRE verify:phone:13800000000 300
TTL verify:phone:13800000000
```

含义：

| 命令 | 说明 |
|------|------|
| `EXPIRE key seconds` | 给 key 设置多少秒后过期 |
| `TTL key` | 查看 key 还剩多少秒 |
| `TTL` 返回 `-1` | key 存在，但没有过期时间 |
| `TTL` 返回 `-2` | key 不存在 |

更常用的写法是写入时直接设置过期时间：

```bash
SET verify:phone:13800000000 "246810" EX 300
```

## 4. 实际应用：缓存一条商品详情

假设数据库里有商品详情，接口经常查询 `product_id=1001`。可以把查询结果暂时放进 Redis：

```bash
SET product:1001 "{\"id\":1001,\"name\":\"机械键盘\",\"price\":299}" EX 600
GET product:1001
```

业务流程：

1. 先查 Redis 的 `product:1001`。
2. 如果查到，直接返回给用户。
3. 如果没查到，再查数据库。
4. 查完数据库后，把结果写回 Redis，并设置过期时间。

这样热门商品详情就不用每次都访问数据库。

## 5. key 命名建议

推荐使用 `业务:对象:id:字段` 的形式：

```text
user:1001:name
user:1001:profile
product:1001:detail
login:token:abc123
verify:phone:13800000000
```

好处是可读性强，也方便定位问题。

## 练习

1. 保存一个 `user:1:email`，值为你的邮箱。
2. 保存一个验证码 key，设置 60 秒过期。
3. 用 `TTL` 观察过期时间变化。
4. 删除一个 key，再用 `GET` 验证是否删除成功。

## 下一章

[第二章：常用数据结构](../chapter02-data-types/) — 学习 String、Hash、List、Set、ZSet 分别适合什么业务场景。

