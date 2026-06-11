package main

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func receiveMessages() error {
	// 1. 连接 RabbitMQ。消费者要连接同一个 RabbitMQ 服务。
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return fmt.Errorf("连接 RabbitMQ 失败：%w", err)
	}
	defer conn.Close()

	// 2. 创建 channel。后续声明队列、注册消费者都通过它完成。
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("创建 channel 失败：%w", err)
	}
	defer ch.Close()

	// 3. 声明队列。这里的队列名要和发送者保持一致。
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

	// 4. 注册消费者。autoAck=true 表示收到消息后自动确认。
	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("注册消费者失败：%w", err)
	}

	// 5. 持续等待消息。for range 会一直阻塞，直到连接关闭。
	fmt.Println("等待消息中，按 Ctrl+C 退出")
	for d := range msgs {
		fmt.Printf("收到消息：%s\n", d.Body)
	}

	return nil
}
