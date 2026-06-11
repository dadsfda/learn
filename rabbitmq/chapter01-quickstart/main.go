package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	// 程序启动时读取命令参数，用来决定是发送消息还是接收消息。
	if len(os.Args) != 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "send":
		err = sendMessage()
	case "receive":
		err = receiveMessages()
	default:
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func printUsage() {
	fmt.Println("用法：go run . [send|receive]")
	fmt.Println("示例：go run . send")
	fmt.Println("示例：go run . receive")
}
