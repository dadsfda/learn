# 第八章：生产实践

> 真正上线 RabbitMQ，重点不是会写 publish/consume，而是知道失败、重复、堆积时怎么办。

## 目标

- 掌握命名规范
- 理解幂等、重试、死信
- 知道监控哪些指标
- 避免常见生产事故

## 1. 命名规范

推荐清晰表达业务：

```text
Exchange:
user.events
order.events
payment.events
notification.direct

Queue:
mail.user_registered
coupon.user_registered
warehouse.order_paid
dlq.order_timeout

Routing Key:
user.registered
order.created
order.paid
payment.failed
```

建议：

- 用小写英文。
- 用点分隔事件层级。
- 队列名体现消费者职责。
- 死信队列统一加 `dlq` 前缀或后缀。

## 2. 消息体设计

推荐事件消息至少包含：

```json
{
  "event_id": "evt_1001",
  "event_type": "order.paid",
  "occurred_at": "2026-06-06T16:00:00+08:00",
  "trace_id": "trace_abc",
  "payload": {
    "order_id": 1001,
    "user_id": 2001
  }
}
```

字段说明：

| 字段 | 作用 |
|------|------|
| `event_id` | 幂等去重 |
| `event_type` | 识别事件类型 |
| `occurred_at` | 事件发生时间 |
| `trace_id` | 链路追踪 |
| `payload` | 业务数据 |

## 3. 消费者幂等

RabbitMQ 常见目标是“至少投递一次”，所以消费者可能收到重复消息。

幂等做法：

- 数据库用唯一索引记录 `event_id`。
- 处理前先判断这条事件是否处理过。
- 业务更新使用状态机，例如只允许 `待支付 -> 已支付`。
- 对外部接口调用要有业务流水号。

示例流程：

```text
收到 order.paid
-> 检查 event_id 是否处理过
-> 未处理则执行业务
-> 记录 event_id 已处理
-> ack
```

## 4. 重试与死信

推荐策略：

```text
可恢复错误 -> 延迟重试
不可恢复错误 -> 进入死信队列
超过最大重试次数 -> 进入死信队列
```

可恢复错误：

- 第三方接口临时超时
- 数据库短暂不可用
- 下游服务重启中

不可恢复错误：

- 消息格式错误
- 必要字段缺失
- 业务对象不存在且无法补偿

## 5. 监控重点

管理后台和监控系统要关注：

| 指标 | 含义 |
|------|------|
| Ready messages | 等待消费的消息数量 |
| Unacked messages | 已投递但未 ack 的消息数量 |
| Publish rate | 消息发布速度 |
| Deliver rate | 消息投递速度 |
| Ack rate | 消费确认速度 |
| Consumer count | 消费者数量 |
| Connection count | 连接数量 |
| Disk free | 磁盘剩余空间 |
| Memory usage | 内存使用 |

风险信号：

- Ready 持续上涨：消费者处理不过来或挂了。
- Unacked 很高：消费者卡住、忘记 ack 或 prefetch 太大。
- 连接数异常上涨：应用连接泄漏。
- 磁盘不足：RabbitMQ 可能触发资源告警并拒绝写入。

## 6. 上线检查清单

上线前确认：

1. exchange、queue、routing key 命名清楚。
2. 重要队列设置 durable。
3. 重要消息设置 persistent。
4. 生产者需要时开启 publisher confirms。
5. 消费者使用手动 ack。
6. 消费者处理逻辑幂等。
7. 失败消息有重试上限。
8. 配置死信队列。
9. 设置合理 prefetch。
10. 监控队列堆积和 unacked。

## 7. 常见坑

| 问题 | 原因 | 建议 |
|------|------|------|
| 消息重复处理 | ack 前消费者崩溃，消息重新投递 | 消费者幂等 |
| 消息堆积 | 消费慢或消费者挂了 | 扩容消费者、优化处理逻辑 |
| Unacked 一直很高 | 忘记 ack 或处理卡住 | 检查消费者日志和 prefetch |
| 消息丢失 | 自动 ack、非持久消息、无发布确认 | 使用可靠性组合 |
| 无限重试 | 一直 nack requeue | 设置重试次数和死信队列 |
| 队列太多 | 每个用户一个队列 | 按业务职责设计队列 |

## 8. RabbitMQ 和 Redis 队列怎么选？

| 需求 | 建议 |
|------|------|
| 简单本地学习、轻量任务 | Redis List/Stream 可以 |
| 复杂路由、广播、主题订阅 | RabbitMQ 更合适 |
| 需要 ack、死信、管理后台 | RabbitMQ 更合适 |
| 超高吞吐日志流、大数据管道 | Kafka 更常见 |
| 已经使用 Redis 且可靠性要求不高 | Redis 更简单 |

## 练习

1. 为 `order.paid` 设计完整 exchange、queue、routing key。
2. 设计一条消息 JSON，包含 `event_id` 和 `trace_id`。
3. 思考消费者如何用数据库唯一索引实现幂等。
4. 打开管理后台，找到 Ready、Unacked、Consumers 这些指标。

## 继续学习

完成本教程后，可以继续学习：

- RabbitMQ quorum queues
- RabbitMQ lazy queues
- RabbitMQ clustering
- RabbitMQ federation 和 shovel
- Spring AMQP 或 Go 项目中的封装实践
- 与 Kafka、Redis Stream 的差异和选型

