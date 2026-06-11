# Redis 小白学习教程

> 从实际应用出发学习 Redis：先会用，再理解为什么这么用。

## 为什么学 Redis？

Redis 是一个基于内存的高性能数据存储，常用于 Web 后端中的“快路径”场景。你不需要一开始就掌握所有内部原理，先记住它最常见的价值：

| 场景 | Redis 解决的问题 |
|------|------------------|
| 缓存 | 减少数据库压力，提高接口响应速度 |
| 登录会话 | 保存 token、验证码、临时状态 |
| 计数器 | 阅读量、点赞数、接口调用次数 |
| 限流 | 防止接口被刷爆 |
| 排行榜 | 游戏积分榜、热度榜、销量榜 |
| 队列 | 异步任务、削峰填谷 |
| 分布式锁 | 多个服务实例之间协调同一件事 |

## 前置条件

- 会使用命令行
- 了解基础 HTTP 接口概念
- 如果要结合 Go 项目练习，建议已经学过 Go 基础和 Gin 基础

## 快速启动 Redis

推荐使用 Docker，最省事：

```bash
docker run --name redis-learn -p 6379:6379 -d redis:7
docker exec -it redis-learn redis-cli
```

进入 `redis-cli` 后测试：

```bash
PING
```

看到下面结果表示 Redis 可用：

```text
PONG
```

停止和删除容器：

```bash
docker stop redis-learn
docker rm redis-learn
```

## 教程目录

| 章节 | 内容 | 关键知识点 |
|------|------|-----------|
| [01 快速入门](chapter01-quickstart/) | 启动 Redis，理解 key/value 和过期时间 | `PING`, `SET`, `GET`, `DEL`, `EXPIRE`, `TTL` |
| [02 常用数据结构](chapter02-data-types/) | String、Hash、List、Set、ZSet 的应用 | 五大基础类型、适用场景 |
| [03 缓存实战](chapter03-cache/) | 用 Redis 缓存商品详情和热点数据 | 缓存命中、失效、穿透、击穿、雪崩 |
| [04 登录会话与验证码](chapter04-session-token/) | 保存 token、短信验证码、一次性状态 | TTL、命名规范、主动删除 |
| [05 计数器与限流](chapter05-counter-rate-limit/) | 阅读量、点赞数、接口访问限制 | `INCR`, `EXPIRE`, 原子操作 |
| [06 排行榜](chapter06-rank-list/) | 积分榜、热度榜、Top N 查询 | `ZADD`, `ZREVRANGE`, `ZRANK` |
| [07 队列与 Stream](chapter07-queue-stream/) | 异步任务和消息消费 | List 队列、Stream 消费组 |
| [08 生产实践](chapter08-production/) | 命名、内存、持久化、监控与风险 | key 设计、淘汰策略、持久化 |

## 学习建议

1. 先把每章命令在 `redis-cli` 里跑一遍。
2. 每学一种数据结构，都问自己：它适合解决什么业务问题？
3. 不要把 Redis 当数据库的替代品，它更适合做缓存、临时状态和高频读写辅助。
4. 每个 key 都要想清楚：谁写入、谁读取、什么时候过期、数据丢了能不能恢复。

## Redis 使用心法

- **能设置过期时间就设置过期时间**：临时数据不要永久占内存。
- **key 命名要有业务含义**：例如 `user:1001:profile`，不要写成 `u1`。
- **缓存不是事实来源**：关键数据仍然以数据库为准。
- **先用简单命令解决问题**：不要一上来就引入复杂 Lua、集群、订阅模式。
- **生产环境谨慎使用全量扫描命令**：比如 `KEYS *`，数据多时会阻塞服务。

