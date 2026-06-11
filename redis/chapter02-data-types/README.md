# 第二章：常用数据结构

> Redis 不只是字符串，它的不同数据结构对应不同业务场景。

## 目标

- 理解 Redis 常用的 5 种基础数据结构
- 能根据业务场景选择合适结构
- 掌握每种结构的核心命令

## 1. String：字符串、数字、JSON

String 是最常用的数据类型，可以存文本、数字、JSON 字符串。

```bash
SET article:1001:title "Redis 入门"
GET article:1001:title

SET article:1001:view_count 0
INCR article:1001:view_count
GET article:1001:view_count
```

适合：

- 缓存 JSON
- 保存 token
- 计数器
- 开关配置

## 2. Hash：对象字段

Hash 适合保存一个对象的多个字段，例如用户资料：

```bash
HSET user:1001 name "小明" age 18 city "杭州"
HGET user:1001 name
HGETALL user:1001
HINCRBY user:1001 age 1
```

适合：

- 用户信息
- 商品摘要
- 配置项集合

注意：如果你总是整块读取对象，String 存 JSON 也可以；如果经常只改某个字段，Hash 更方便。

## 3. List：有顺序的列表

List 适合做简单队列、最近记录：

```bash
LPUSH tasks "send_email:1001"
LPUSH tasks "send_sms:1002"
RPOP tasks
```

也可以保存最近浏览：

```bash
LPUSH user:1001:recent_products 2001
LPUSH user:1001:recent_products 2002
LRANGE user:1001:recent_products 0 9
LTRIM user:1001:recent_products 0 9
```

适合：

- 简单任务队列
- 最近浏览
- 最新消息列表

## 4. Set：不重复集合

Set 会自动去重，适合点赞用户、标签、共同好友：

```bash
SADD article:1001:liked_users 1001
SADD article:1001:liked_users 1002
SADD article:1001:liked_users 1001
SCARD article:1001:liked_users
SISMEMBER article:1001:liked_users 1001
```

适合：

- 点赞去重
- 抽奖参与用户
- 标签集合
- 共同关注、共同好友

## 5. ZSet：带分数的有序集合

ZSet 每个成员都有一个分数，适合排行榜：

```bash
ZADD game:rank 9800 user:1001
ZADD game:rank 7600 user:1002
ZADD game:rank 12000 user:1003

ZREVRANGE game:rank 0 2 WITHSCORES
ZREVRANK game:rank user:1001
ZSCORE game:rank user:1001
```

适合：

- 积分榜
- 热搜榜
- 销量榜
- 内容热度榜

## 6. 选型速查

| 需求 | 推荐类型 | 示例 key |
|------|----------|----------|
| 缓存一个 JSON | String | `product:1001:detail` |
| 保存用户多个字段 | Hash | `user:1001` |
| 最近浏览 10 个商品 | List | `user:1001:recent_products` |
| 文章点赞用户去重 | Set | `article:1001:liked_users` |
| 游戏积分排行榜 | ZSet | `game:rank` |

## 实际应用：文章点赞

需求：

- 用户只能点赞一次
- 要能判断用户是否已点赞
- 要能统计点赞数

使用 Set：

```bash
SADD article:1001:liked_users user:1001
SISMEMBER article:1001:liked_users user:1001
SCARD article:1001:liked_users
SREM article:1001:liked_users user:1001
```

为什么不用简单数字计数？因为数字只能知道有多少赞，不能判断某个用户是否已经点过。

## 练习

1. 用 Hash 保存一个商品的 `name`、`price`、`stock`。
2. 用 List 保存用户最近浏览的 5 个商品。
3. 用 Set 实现文章收藏去重。
4. 用 ZSet 创建一个分数排行榜，并查询前 3 名。

## 下一章

[第三章：缓存实战](../chapter03-cache/) — 学习 Redis 最常见的缓存用法和常见坑。

