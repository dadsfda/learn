# 第七章：队列与 Stream

> Redis 可以做轻量级队列；简单任务用 List，更可靠的消息消费用 Stream。

## 目标

- 使用 List 实现简单队列
- 理解阻塞消费
- 初步了解 Stream 和消费组

## 1. List 简单队列

生产任务：

```bash
LPUSH queue:email "send_welcome_email:user:1001"
LPUSH queue:email "send_reset_email:user:1002"
```

消费任务：

```bash
RPOP queue:email
```

这样就形成了先进先出的队列。

## 2. 阻塞消费

普通 `RPOP` 在没有任务时会直接返回空。后台 worker 更常用阻塞消费：

```bash
BRPOP queue:email 10
```

含义：

- 如果队列有任务，立即返回。
- 如果没有任务，最多等待 10 秒。
- 超时后返回空。

## 3. List 队列适合什么

适合：

- 个人项目
- 简单异步任务
- 任务丢失影响不大的场景
- 单消费者或少量消费者

不适合：

- 强可靠消息
- 复杂重试
- 消费确认
- 消息追踪

如果业务对消息可靠性要求很高，应该优先考虑专业消息队列，例如 Kafka、RabbitMQ、RocketMQ。

## 4. Stream 入门

Stream 是 Redis 提供的更完整的消息流结构。

写入消息：

```bash
XADD stream:orders * order_id 1001 user_id 2001 amount 99
XADD stream:orders * order_id 1002 user_id 2002 amount 199
```

读取消息：

```bash
XRANGE stream:orders - +
```

从最新位置读取新消息：

```bash
XREAD COUNT 2 STREAMS stream:orders 0
```

## 5. 消费组

创建消费组：

```bash
XGROUP CREATE stream:orders group:order-workers 0 MKSTREAM
```

消费者读取：

```bash
XREADGROUP GROUP group:order-workers worker-1 COUNT 1 STREAMS stream:orders >
```

确认处理完成：

```bash
XACK stream:orders group:order-workers 1700000000000-0
```

其中消息 ID 需要替换成实际读取到的 ID。

## 6. 实际应用：下单后异步处理

下单接口里不要同步做所有事情，可以把非核心任务放到队列：

```text
用户下单
-> 写订单数据库
-> Redis Stream 写入 order_created 消息
-> 立即返回下单成功
-> 后台 worker 消费消息
-> 发送短信、发放积分、通知仓库
```

这样接口响应更快，也能让后续任务独立处理。

## 7. Go 伪代码：写入订单消息

```go
func PublishOrderCreated(orderID int64, userID int64) error {
    values := map[string]interface{}{
        "event": "order_created",
        "order_id": orderID,
        "user_id": userID,
    }

    return redis.XAdd(ctx, &redis.XAddArgs{
        Stream: "stream:orders",
        Values: values,
    }).Err()
}
```

## 练习

1. 用 List 创建一个邮件发送队列。
2. 用 `BRPOP` 模拟 worker 等待任务。
3. 用 Stream 写入两条订单消息。
4. 创建消费组并读取一条消息。

## 下一章

[第八章：生产实践](../chapter08-production/) — 学习生产环境中 Redis 的关键注意事项。

