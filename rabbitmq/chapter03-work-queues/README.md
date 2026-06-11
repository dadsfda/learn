# 第三章：工作队列

> 工作队列用于把耗时任务交给后台 Worker，并让多个 Worker 一起分担压力。

## 目标

- 理解 Work Queue 模式
- 使用多个消费者分担任务
- 理解手动 ack 和 prefetch 的价值

## 1. 适用场景

Web 请求里不适合做太慢的事情，例如：

- 发送邮件
- 生成 PDF
- 压缩图片
- 批量导入数据
- 调用慢速第三方接口

更好的方式：

```text
HTTP 接口 -> 写任务消息 -> 立即返回
后台 Worker -> 慢慢消费任务
```

## 2. 生产任务

把任务写到 `task_queue`：

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

发布消息时设置持久化：

```go
err = ch.PublishWithContext(
    ctx,
    "",
    q.Name,
    false,
    false,
    amqp.Publishing{
        DeliveryMode: amqp.Persistent,
        ContentType:  "text/plain",
        Body:         []byte("send_email:user:1001"),
    },
)
```

## 3. 多个 Worker 分担任务

同时启动两个消费者：

```bash
go run worker.go
go run worker.go
```

再连续发送多条任务：

```bash
go run new_task.go "task 1"
go run new_task.go "task 2"
go run new_task.go "task 3"
go run new_task.go "task 4"
```

RabbitMQ 会把消息分发给不同 Worker。这个模式也叫竞争消费者：多个消费者监听同一个队列，一条消息只会被其中一个消费者处理。

## 4. 手动 ack

不要在重要任务里使用自动 ack。

自动 ack 的问题：

```text
RabbitMQ 把消息发给消费者
-> 立即认为消息处理成功
-> 消费者处理到一半崩溃
-> 消息丢失
```

手动 ack 的流程：

```text
RabbitMQ 投递消息
-> 消费者处理业务
-> 处理成功后 d.Ack(false)
-> RabbitMQ 删除消息
```

消费代码核心：

```go
msgs, err := ch.Consume(
    q.Name,
    "",
    false,
    false,
    false,
    false,
    nil,
)

for d := range msgs {
    err := handleTask(d.Body)
    if err != nil {
        d.Nack(false, true)
        continue
    }
    d.Ack(false)
}
```

`Nack(false, true)` 表示当前消息处理失败，并重新放回队列。

## 5. prefetch：不要一次塞太多任务给一个 Worker

如果一个 Worker 很慢，RabbitMQ 仍然一次发很多消息给它，就会造成任务堆在慢 Worker 内存里。

可以设置 prefetch：

```go
err = ch.Qos(
    1,
    0,
    false,
)
```

含义：当前消费者同一时间最多处理 1 条未 ack 的消息。处理完一条并 ack 后，RabbitMQ 再发下一条。

## 实际应用：图片压缩任务

```text
用户上传图片
-> 上传接口保存原图
-> 发送 compress_image 消息
-> 接口返回上传成功
-> Worker 消费消息，生成缩略图
-> 成功后 ack
-> 失败后 nack 或进入死信队列
```

消息内容建议：

```json
{
  "task_id": "task_1001",
  "type": "compress_image",
  "image_id": 9527,
  "source_path": "/uploads/raw/9527.png"
}
```

## 练习

1. 设计一个 `queue:email` 队列处理邮件任务。
2. 思考：消费者崩溃时，为什么手动 ack 更安全？
3. 把 prefetch 设置为 `1`，观察多个 Worker 的任务分配。
4. 思考：如果任务一直失败并反复 requeue，会发生什么？

## 下一章

[第四章：发布订阅](../chapter04-pub-sub/) — 学习一条消息广播给多个消费者。

