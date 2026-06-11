# 第五章：路由与主题

> direct 用于精确匹配，topic 用于模式匹配。它们都能让消费者只收到自己关心的消息。

## 目标

- 使用 direct exchange 按 routing key 精确路由
- 使用 topic exchange 按模式订阅
- 能设计日志、订单、通知类路由

## 1. direct：精确路由

声明 direct exchange：

```go
err := ch.ExchangeDeclare(
    "direct_logs",
    "direct",
    true,
    false,
    false,
    false,
    nil,
)
```

发布错误日志：

```go
err = ch.PublishWithContext(
    ctx,
    "direct_logs",
    "error",
    false,
    false,
    amqp.Publishing{
        ContentType: "text/plain",
        Body:        []byte("database timeout"),
    },
)
```

消费者绑定自己关心的级别：

```go
err = ch.QueueBind(
    q.Name,
    "error",
    "direct_logs",
    false,
    nil,
)
```

如果队列绑定了 `error`，就只会收到 routing key 为 `error` 的消息。

## 2. direct 适合什么

适合分类明确、匹配规则简单的场景：

```text
log.error
log.warn
log.info

sms
email
push

order.created
order.paid
order.cancelled
```

## 3. topic：模式匹配

topic exchange 的 routing key 通常用点分隔：

```text
order.created
order.paid
user.registered
payment.failed
```

匹配符：

| 符号 | 含义 |
|------|------|
| `*` | 匹配一个单词 |
| `#` | 匹配零个或多个单词 |

示例：

```text
order.*      匹配 order.created、order.paid
*.failed     匹配 payment.failed、sms.failed
order.#      匹配 order.created、order.payment.failed
#            匹配所有消息
```

## 4. topic 示例

声明 topic exchange：

```go
err := ch.ExchangeDeclare(
    "biz.events",
    "topic",
    true,
    false,
    false,
    false,
    nil,
)
```

绑定订单相关事件：

```go
err = ch.QueueBind(
    q.Name,
    "order.*",
    "biz.events",
    false,
    nil,
)
```

发布事件：

```go
err = ch.PublishWithContext(
    ctx,
    "biz.events",
    "order.created",
    false,
    false,
    amqp.Publishing{
        ContentType: "application/json",
        Body:        []byte(`{"order_id":1001}`),
    },
)
```

## 实际应用：多业务事件中心

设计一个业务事件 exchange：

```text
Exchange: biz.events
Type: topic
```

事件：

```text
user.registered
user.deleted
order.created
order.paid
order.cancelled
payment.failed
sms.failed
```

订阅者：

```text
analytics.queue  -> 绑定 #
order.queue      -> 绑定 order.*
alert.queue      -> 绑定 *.failed
coupon.queue     -> 绑定 user.registered
```

这样一个 exchange 就能支撑多个业务订阅关系。

## 选型建议

| 需求 | 推荐 |
|------|------|
| 所有订阅者都收一份 | fanout |
| 按固定类别精确分发 | direct |
| 按多级业务事件匹配 | topic |
| 只是后台任务分摊 | 普通队列 |

## 练习

1. 设计一个 `notify.direct`，按 `sms`、`email`、`push` 路由。
2. 设计一个 `app.events`，让消费者订阅所有 `order.*` 事件。
3. 思考：`#` 绑定有什么风险？
4. 思考：什么时候 direct 比 topic 更简单？

## 下一章

[第六章：确认与持久化](../chapter06-ack-durability/) — 学习如何降低消息丢失风险。

