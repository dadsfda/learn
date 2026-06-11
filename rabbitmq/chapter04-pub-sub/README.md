# 第四章：发布订阅

> 发布订阅用于把同一条消息广播给多个消费者，每个消费者都能收到一份。

## 目标

- 理解 fanout exchange
- 理解临时队列和绑定
- 能设计一个事件广播场景

## 1. Work Queue 和 Pub/Sub 的区别

| 模式 | 消息会被谁收到 | 适合场景 |
|------|----------------|----------|
| Work Queue | 多个 Worker 中的一个 | 任务分摊，例如发邮件、压缩图片 |
| Pub/Sub | 每个订阅者都收到一份 | 事件广播，例如用户注册、订单创建 |

Work Queue 强调“分工处理同一批任务”。

Pub/Sub 强调“多个系统各自响应同一个事件”。

## 2. 声明 fanout exchange

fanout 会把消息广播给所有绑定到它的队列：

```go
err := ch.ExchangeDeclare(
    "logs",
    "fanout",
    true,
    false,
    false,
    false,
    nil,
)
```

发布消息：

```go
err = ch.PublishWithContext(
    ctx,
    "logs",
    "",
    false,
    false,
    amqp.Publishing{
        ContentType: "text/plain",
        Body:        []byte("user 1001 registered"),
    },
)
```

fanout 不关心 routing key，所以这里可以传空字符串。

## 3. 绑定临时队列

每个消费者创建自己的队列，并绑定到 exchange：

```go
q, err := ch.QueueDeclare(
    "",
    false,
    false,
    true,
    false,
    nil,
)
```

空队列名表示让 RabbitMQ 自动生成队列名。`exclusive=true` 表示连接断开后队列会被删除，适合临时订阅者。

绑定：

```go
err = ch.QueueBind(
    q.Name,
    "",
    "logs",
    false,
    nil,
)
```

然后消费这个队列：

```go
msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
```

## 实际应用：用户注册事件

用户服务只发布一个事件：

```text
Exchange: user.events
Type: fanout
Message: user.registered
```

多个消费者各自绑定队列：

```text
mail.user_registered       -> 发送欢迎邮件
coupon.user_registered     -> 发新人优惠券
analytics.user_registered  -> 统计注册数据
```

好处：

- 用户服务不需要知道有哪些下游系统。
- 新增一个消费者不需要改用户服务。
- 某个下游失败不会直接拖慢注册接口。

## 什么时候不用 fanout？

如果所有消费者都要收到同一份消息，用 fanout。

如果不同消费者只关心部分消息，用 direct 或 topic。比如日志系统中，错误日志要写文件，普通日志只打印控制台，就不适合全部广播。

## 练习

1. 设计一个 `order.events` 的 fanout exchange。
2. 给它绑定两个队列：一个发短信，一个加积分。
3. 思考：如果某个消费者短暂下线，是否还应该保留它的队列？
4. 思考：临时队列和持久队列分别适合什么订阅者？

## 下一章

[第五章：路由与主题](../chapter05-routing-topics/) — 学习按条件选择性消费消息。

