# RabbitMQ 消息队列小白学习教程

> 以 RabbitMQ 为主线，从“为什么需要消息队列”学到“如何在项目里可靠使用消息队列”。

## 参考资料

- [RabbitMQ 官方教程](https://www.rabbitmq.com/tutorials)：官方教程覆盖 Hello World、Work Queues、Publish/Subscribe、Routing、Topics、RPC、Publisher Confirms 等基础模式。
- [RabbitMQ Consumer Acknowledgements and Publisher Confirms](https://www.rabbitmq.com/docs/confirms)：官方可靠性文档，重点讲消费者确认、发布者确认和 prefetch。
- [RabbitMQ Queues](https://www.rabbitmq.com/docs/queues)：队列概念和参数说明。
- [RabbitMQ Exchanges](https://www.rabbitmq.com/docs/exchanges)：交换机类型和路由模型说明。

## 什么是消息队列？

消息队列可以理解成“系统之间的缓冲区”。生产者把消息交给队列，消费者从队列里取消息处理。它常用来解决这些问题：

| 场景 | 直接调用的问题 | 使用消息队列后的效果 |
|------|----------------|----------------------|
| 发送邮件/短信 | 用户要等第三方接口返回 | 接口先返回，后台慢慢发 |
| 下单后处理 | 扣库存、发积分、通知仓库都挤在请求里 | 核心链路更短，非核心任务异步做 |
| 高峰流量 | 瞬时请求打爆下游系统 | 队列削峰，下游按能力消费 |
| 多系统通知 | 一个服务要调用多个服务 | 发布一条事件，多个消费者订阅 |
| 日志采集 | 应用直接写远程日志容易阻塞 | 先写队列，再集中处理 |

## 为什么选择 RabbitMQ？

- 支持成熟的 AMQP 0-9-1 协议，生态完善。
- 交换机模型强大，能做广播、精确路由、模式匹配路由。
- 支持消息确认、持久化、死信队列、发布确认等可靠性能力。
- 管理后台直观，适合学习和排查问题。
- 很适合中小型后端系统、业务事件分发、异步任务处理。

## 快速启动

推荐使用带管理后台的 Docker 镜像：

```bash
docker run -d --name rabbitmq-learn `
  -p 5672:5672 `
  -p 15672:15672 `
  rabbitmq:4-management
```

访问管理后台：

```text
http://localhost:15672
```

默认账号密码：

```text
guest / guest
```

AMQP 连接地址：

```text
amqp://guest:guest@localhost:5672/
```

停止和删除容器：

```bash
docker stop rabbitmq-learn
docker rm rabbitmq-learn
```

## 教程目录

| 章节 | 内容 | 关键知识点 |
|------|------|-----------|
| [01 快速入门](chapter01-quickstart/) | 启动 RabbitMQ，理解生产者和消费者 | Docker、管理后台、Go 客户端 |
| [02 核心概念](chapter02-core-concepts/) | Broker、Queue、Exchange、Binding、Routing Key | 消息流转模型 |
| [03 工作队列](chapter03-work-queues/) | 多个 Worker 分担耗时任务 | Work Queue、竞争消费者、prefetch |
| [04 发布订阅](chapter04-pub-sub/) | 一条消息广播给多个消费者 | fanout exchange、临时队列 |
| [05 路由与主题](chapter05-routing-topics/) | 按日志级别、业务类型选择性消费 | direct、topic、binding key |
| [06 确认与持久化](chapter06-ack-durability/) | 尽量不丢消息 | ack、nack、durable、persistent、publisher confirms |
| [07 RPC、延迟与死信](chapter07-rpc-delay-dlx/) | 请求响应、延迟任务、失败兜底 | reply queue、TTL、DLX |
| [08 生产实践](chapter08-production/) | 上线前检查清单 | 命名、幂等、重试、监控、风险 |

## 学习建议

1. 先用管理后台观察队列、交换机、消息数量变化。
2. 每学一个模式，都想一个真实业务场景套进去。
3. 初学时不要急着追求“绝对不丢消息”，先理解消息从哪里到哪里。
4. 真正写项目时，消费者必须考虑幂等，因为消息可能重复投递。

## 消息队列使用心法

- **队列不是数据库**：不要把队列当长期数据存储。
- **异步意味着最终一致**：用户看到的结果可能会晚一点完成。
- **消费者要幂等**：同一条消息重复消费，结果也应该正确。
- **失败要有去处**：处理失败的消息要重试、丢弃或进入死信队列。
- **可靠性是组合拳**：生产确认、队列持久化、消息持久化、消费者 ack 都要一起考虑。

