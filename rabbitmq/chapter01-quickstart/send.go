package main

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func sendMessage() error {
	// 1. 连接 RabbitMQ。这里使用 Docker 容器默认的账号、密码和端口。
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return fmt.Errorf("连接 RabbitMQ 失败：%w", err)
	}
	defer conn.Close()

	// 2. 创建 channel。大多数 RabbitMQ 操作都通过 channel 完成。
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("创建 channel 失败：%w", err)
	}
	defer ch.Close()

	// 3. 声明队列。发送者和消费者声明同一个队列，能保证队列存在。
	q, err := ch.QueueDeclare(
		"hello",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("声明队列失败：%w", err)
	}

	// 4. 设置发送超时时间，避免网络异常时程序一直卡住。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 5. 使用默认交换机发送消息。routing key 写队列名，消息会进入这个队列。
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
	if err != nil {
		return fmt.Errorf("发送消息失败：%w", err)
	}


	fmt.Printf("发送成功：%s\n", body)
	return nil
}
