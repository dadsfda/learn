# 第一章：快速入门

> 先跑起来：启动 RabbitMQ，进入管理后台，理解最小生产者和消费者。

## 目标

- 使用 Docker 启动 RabbitMQ
- 认识管理后台
- 用 Go 客户端发送和消费一条消息

## 1. 启动 RabbitMQ

```bash
docker run -d --name rabbitmq-learn `
  -p 5672:5672 `
  -p 15672:15672 `
  rabbitmq:4-management
```

端口说明：

| 端口 | 作用 |
|------|------|
| `5672` | 应用连接 RabbitMQ 的 AMQP 端口 |
| `15672` | Web 管理后台端口 |

打开后台：

```text
http://localhost:15672
```

默认账号密码：

```text
guest / guest
```

## 2. 创建 Go 示例项目

```bash
mkdir rabbitmq-demo
cd rabbitmq-demo
go mod init rabbitmq-demo
go get github.com/rabbitmq/amqp091-go
```

官方 Go 教程使用的是 `github.com/rabbitmq/amqp091-go`，它是 RabbitMQ 的 AMQP 0-9-1 Go 客户端。

## 3. 发送消息

创建 `send.go`：

```go
package main

import (
    "context"
    "log"
    "time"

    amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
    if err != nil {
        log.Panicf("%s: %s", msg, err)
    }
}

func main() {
    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
    failOnError(err, "连接 RabbitMQ 失败")
    defer conn.Close()

    ch, err := conn.Channel()
    failOnError(err, "打开 channel 失败")
    defer ch.Close()

    q, err := ch.QueueDeclare(
        "hello",
        true,
        false,
        false,
        false,
        nil,
    )
    failOnError(err, "声明队列失败")

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    body := "Hello RabbitMQ"
    err = ch.PublishWithContext(
        ctx,
        "",
        q.Name,
        false,
        false,
        amqp.Publishing{
            ContentType: "text/plain",
            Body:        []byte(body),
        },
    )
    failOnError(err, "发布消息失败")

    log.Printf("发送成功: %s", body)
}
```

运行：

```bash
go run . send
```

## 4. 消费消息

创建 `receive.go`：

```go
package main

import (
    "log"

    amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
    if err != nil {
        log.Panicf("%s: %s", msg, err)
    }
}

func main() {
    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
    failOnError(err, "连接 RabbitMQ 失败")
    defer conn.Close()

    ch, err := conn.Channel()
    failOnError(err, "打开 channel 失败")
    defer ch.Close()

    q, err := ch.QueueDeclare(
        "hello",
        true,
        false,
        false,
        false,
        nil,
    )
    failOnError(err, "声明队列失败")

    msgs, err := ch.Consume(
        q.Name,
        "",
        true,
        false,
        false,
        false,
        nil,
    )
    failOnError(err, "注册消费者失败")

    log.Println("等待消息，按 Ctrl+C 退出")
    for d := range msgs {
        log.Printf("收到消息: %s", d.Body)
    }
}
```

运行：

```bash
go run . receive
```

再开一个终端运行：

```bash
go run . send
```

消费者会打印收到的消息。

## 5. 在管理后台观察

进入管理后台后，可以查看：

- `Queues and Streams`：队列列表和消息数量
- `Exchanges`：交换机列表
- `Connections`：当前连接
- `Channels`：连接内部的通道

如果先运行 `send.go`，再进入后台查看 `hello` 队列，可以看到队列里有消息；运行 `receive.go` 后，消息会被消费掉。

## 实际应用：异步发送欢迎邮件

用户注册接口可以只做核心事情：

```text
写入用户表
-> 发送 user_registered 消息到队列
-> 立即返回注册成功
-> 后台消费者发送欢迎邮件
```

这样用户不用等待邮件服务返回。

## 练习

1. 修改消息内容，再发送一次。
2. 先发送 3 条消息，再启动消费者，观察消费顺序。
3. 在管理后台查看 `hello` 队列的消息数量变化。
4. 停止 RabbitMQ 容器后再运行发送程序，观察错误信息。

## 下一章

[第二章：核心概念](../chapter02-core-concepts/) — 学习消息在 RabbitMQ 中如何流转。
