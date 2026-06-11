# 第二章：核心概念

> 学 RabbitMQ 先抓住一条线：生产者把消息发给交换机，交换机根据规则把消息投递到队列，消费者从队列取消息。

## 目标

- 理解 Broker、Producer、Consumer
- 理解 Queue、Exchange、Binding、Routing Key
- 能画出一条消息的流转路径

## 1. 最小模型

```text
Producer -> Queue -> Consumer
```

这是第一章的模型。生产者直接把消息发到默认交换机，默认交换机根据 routing key 把消息送到同名队列。

代码里的关键参数：

```go
ch.PublishWithContext(ctx, "", q.Name, false, false, publishing)
```

含义：

| 参数 | 示例 | 说明 |
|------|------|------|
| exchange | `""` | 空字符串表示默认交换机 |
| routing key | `hello` | 默认交换机会把消息发到同名队列 |
| body | `Hello RabbitMQ` | 真正的消息内容 |

## 2. 完整模型

```text
Producer -> Exchange -> Binding -> Queue -> Consumer
```

核心对象：

| 概念 | 说明 |
|------|------|
| Broker | RabbitMQ 服务本身 |
| Producer | 发送消息的应用 |
| Consumer | 接收消息的应用 |
| Queue | 保存消息的队列 |
| Exchange | 接收生产者消息，并决定发到哪些队列 |
| Binding | 交换机和队列之间的绑定关系 |
| Routing Key | 生产者发布消息时带上的路由标识 |
| Binding Key | 队列绑定交换机时使用的匹配规则 |

## 3. 为什么需要 Exchange？

如果只有队列，生产者必须知道消息要进哪个队列。

有了交换机后，生产者只关心“我发布了什么事件”，至于谁订阅、订阅几份，由交换机和绑定决定。

例如：

```text
订单服务发布 order.created
-> 积分服务消费，给用户加积分
-> 短信服务消费，发送下单短信
-> 仓库服务消费，准备发货
```

订单服务不需要直接调用三个服务。

## 4. 常见交换机类型

| 类型 | 路由方式 | 常见场景 |
|------|----------|----------|
| direct | routing key 精确匹配 binding key | 按日志级别、业务类型精确路由 |
| fanout | 广播到所有绑定队列 | 发布订阅、广播通知 |
| topic | 按模式匹配 routing key | 多维度事件订阅 |
| headers | 按消息头匹配 | 不常用，初学可跳过 |

## 5. 队列参数先记这些

声明队列时常见参数：

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

| 参数 | 说明 |
|------|------|
| name | 队列名称 |
| durable | RabbitMQ 重启后队列是否还在 |
| autoDelete | 没有消费者后是否自动删除 |
| exclusive | 是否只给当前连接使用 |
| noWait | 是否不等待服务端确认 |
| args | 扩展参数，例如死信交换机、队列类型 |

小白阶段建议：

- 业务队列通常 `durable=true`。
- 临时广播队列可以让 RabbitMQ 自动命名，并设置 `exclusive=true`。
- 不确定时先不要乱加复杂参数。

## 实际应用：订单创建事件

设计：

```text
Exchange: order.events
Routing Key: order.created
Queue 1: points.order_created
Queue 2: sms.order_created
Queue 3: warehouse.order_created
```

消息内容：

```json
{
  "event_id": "evt_1001",
  "event_type": "order.created",
  "order_id": 1001,
  "user_id": 2001,
  "created_at": "2026-06-06T16:00:00+08:00"
}
```

注意：消息里最好带 `event_id`，消费者可以用它做幂等处理。

## 练习

1. 用自己的话解释 Producer、Exchange、Queue、Consumer 的关系。
2. 思考：发送短信任务适合直接发队列，还是发布事件？
3. 设计一个 `user.registered` 事件的 exchange、routing key 和队列名称。

## 下一章

[第三章：工作队列](../chapter03-work-queues/) — 学习多个消费者分担耗时任务。

