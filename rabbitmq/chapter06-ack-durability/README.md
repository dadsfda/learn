# 第六章：确认与持久化

> 可靠消息不是一个开关，而是一组机制配合：生产者确认、队列持久化、消息持久化、消费者 ack。

## 目标

- 理解消费者 ack
- 理解队列和消息持久化
- 理解 publisher confirms
- 知道哪些情况仍然可能重复或失败

## 1. 消费者确认 ack

自动 ack：

```go
msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
```

`autoAck=true` 表示 RabbitMQ 投递后就认为处理完成。消费者崩溃时，消息可能丢失。

手动 ack：

```go
msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
```

处理成功：

```go
d.Ack(false)
```

处理失败并重新入队：

```go
d.Nack(false, true)
```

处理失败并丢弃，或进入死信队列：

```go
d.Nack(false, false)
```

## 2. 队列持久化

队列持久化让队列定义在 RabbitMQ 重启后仍然存在：

```go
q, err := ch.QueueDeclare(
    "task_queue",
    true,
    false,
    false,
    false,
    nil,
)
```

第二个参数 `durable=true`。

注意：如果一个队列已经以非持久方式创建，不能用同名队列改成持久化。需要删除旧队列或换新名称。

## 3. 消息持久化

发布消息时设置：

```go
amqp.Publishing{
    DeliveryMode: amqp.Persistent,
    ContentType:  "application/json",
    Body:         body,
}
```

队列持久化和消息持久化要一起设置。只设置一个不够。

## 4. 发布者确认 Publisher Confirms

消息从生产者发出，不代表 RabbitMQ 已经可靠接收。发布者确认用于让 RabbitMQ 告诉生产者：消息已经被 broker 接收处理。

开启确认模式：

```go
err := ch.Confirm(false)
```

发布后等待确认：

```go
confirm := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

err = ch.PublishWithContext(ctx, exchange, routingKey, false, false, publishing)
if err != nil {
    return err
}

confirmed := <-confirm
if !confirmed.Ack {
    return errors.New("message was not confirmed")
}
```

初学时先理解：生产者要想知道消息是否真正到达 RabbitMQ，就需要 publisher confirms。

## 5. prefetch 和公平分发

消费者处理慢时，限制未确认消息数量：

```go
err = ch.Qos(1, 0, false)
```

这样 RabbitMQ 不会一次给某个消费者塞太多消息。

## 6. 可靠性组合

| 风险 | 应对 |
|------|------|
| 生产者发出后网络断开 | publisher confirms |
| RabbitMQ 重启导致队列丢失 | durable queue |
| RabbitMQ 重启导致消息丢失 | persistent message |
| 消费者处理到一半崩溃 | manual ack |
| 消息重复投递 | 消费者幂等 |
| 消息一直失败 | 重试次数 + 死信队列 |

## 实际应用：支付成功后发货

支付成功事件非常重要，建议：

```text
生产者开启 publisher confirms
-> 发布 payment.paid 消息
-> exchange、queue 都持久化
-> message 设置 persistent
-> 消费者处理成功后 ack
-> 消费者按 payment_id 做幂等
-> 多次失败后进入死信队列
```

即使这样，也不要假设消息“绝对只处理一次”。更现实的目标是：至少处理一次 + 消费者幂等。

## 练习

1. 把第一章的消费者改成手动 ack。
2. 声明一个持久化队列 `task_queue`。
3. 发布持久化消息。
4. 思考：为什么消息队列系统通常要求消费者幂等？

## 下一章

[第七章：RPC、延迟与死信](../chapter07-rpc-delay-dlx/) — 学习几个常见高级用法。

