package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// ChatEvent 表示 A2UI 协议的聊天事件
type ChatEvent struct {
	Type    string `json:"type"`    // 事件类型：text, error, status
	Content string `json:"content"` // 事件内容
	Done    bool   `json:"done"`    // 是否结束
}

func main() {
	// 检查命令行参数
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "sse":
			runSSEServer()
		case "json-sse":
			runJSONSSEServer()
		case "chat":
			runChatServer()
		default:
			printUsage()
		}
	} else {
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Eino A2UI Protocol 示例程序")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  go run main.go sse       - 运行基础 SSE 服务")
	fmt.Println("  go run main.go json-sse  - 运行 JSON 格式 SSE 服务")
	fmt.Println("  go run main.go chat      - 运行完整的聊天服务（需要 OPENAI_API_KEY）")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  go run main.go sse")
	fmt.Println("  # 然后在浏览器中访问 http://localhost:8080")
}

// runSSEServer 运行基础 SSE 服务
func runSSEServer() {
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		// 设置 SSE 响应头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// 获取 Flusher 用于立即发送数据
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// 发送 SSE 事件
		for i := 0; i < 10; i++ {
			fmt.Fprintf(w, "data: 消息 %d - %s\n\n", i, time.Now().Format("15:04:05"))
			flusher.Flush()
			time.Sleep(time.Second)
		}

		// 发送结束事件
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	})

	// 提供简单的前端页面
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>基础 SSE 示例</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; }
        #messages { background: #f5f5f5; padding: 20px; border-radius: 8px; min-height: 200px; }
        .message { margin: 5px 0; padding: 5px; background: white; border-radius: 4px; }
    </style>
</head>
<body>
    <h1>基础 SSE 示例</h1>
    <button onclick="startSSE()">开始接收事件</button>
    <div id="messages"></div>

    <script>
        function startSSE() {
            const messages = document.getElementById('messages');
            messages.innerHTML = '';

            const eventSource = new EventSource('/events');

            eventSource.onmessage = function(event) {
                if (event.data === '[DONE]') {
                    messages.innerHTML += '<div class="message"><strong>完成</strong></div>';
                    eventSource.close();
                } else {
                    messages.innerHTML += '<div class="message">' + event.data + '</div>';
                }
            };

            eventSource.onerror = function() {
                messages.innerHTML += '<div class="message" style="color:red;">连接错误</div>';
                eventSource.close();
            };
        }
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	})

	fmt.Println("基础 SSE 服务启动在 http://localhost:8080")
	fmt.Println("访问 http://localhost:8080 查看示例")
	http.ListenAndServe(":8080", nil)
}

// runJSONSSEServer 运行 JSON 格式 SSE 服务
func runJSONSSEServer() {
	http.HandleFunc("/api/stream", func(w http.ResponseWriter, r *http.Request) {
		// 设置 SSE 响应头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// 模拟流式响应
		texts := []string{"你", "好", "，", "我", "是", "AI", "助", "手", "！", "有", "什", "么", "可", "以", "帮", "助", "你", "的", "吗", "？"}
		for _, text := range texts {
			event := ChatEvent{
				Type:    "text",
				Content: text,
				Done:    false,
			}

			jsonData, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()

			time.Sleep(100 * time.Millisecond)
		}

		// 发送完成事件
		event := ChatEvent{
			Type:    "text",
			Content: "",
			Done:    true,
		}
		jsonData, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	})

	// 提供前端页面
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>JSON SSE 示例</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; }
        #messages { background: #f5f5f5; padding: 20px; border-radius: 8px; min-height: 200px; white-space: pre-wrap; }
    </style>
</head>
<body>
    <h1>JSON SSE 示例</h1>
    <button onclick="startStream()">开始流式传输</button>
    <div id="messages"></div>

    <script>
        function startStream() {
            const messages = document.getElementById('messages');
            messages.textContent = '';

            const eventSource = new EventSource('/api/stream');

            eventSource.onmessage = function(event) {
                const data = JSON.parse(event.data);

                if (data.done) {
                    messages.textContent += '\n\n[完成]';
                    eventSource.close();
                } else {
                    messages.textContent += data.content;
                }
            };

            eventSource.onerror = function() {
                messages.textContent += '\n[连接错误]';
                eventSource.close();
            };
        }
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	})

	fmt.Println("JSON SSE 服务启动在 http://localhost:8080")
	fmt.Println("访问 http://localhost:8080 查看示例")
	http.ListenAndServe(":8080", nil)
}

// runChatServer 运行完整的聊天服务
func runChatServer() {
	ctx := context.Background()

	// 检查 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("错误: 请设置 OPENAI_API_KEY 环境变量")
		fmt.Println("  export OPENAI_API_KEY=\"your-api-key-here\"")
		os.Exit(1)
	}

	// 创建 ChatModel
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:  "gpt-4o",
		APIKey: apiKey,
	})
	if err != nil {
		fmt.Printf("创建 ChatModel 失败: %v\n", err)
		os.Exit(1)
	}

	// 注册聊天 API
	http.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		// 设置 SSE 响应头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// 获取用户消息
		message := r.URL.Query().Get("message")
		if message == "" {
			sendErrorEvent(w, flusher, "消息不能为空")
			return
		}

		// 构建消息列表
		messages := []*model.Message{
			model.SystemMessage("你是一个 helpful 的助手，请用简洁明了的语言回答问题。"),
			model.UserMessage(message),
		}

		// 使用流式接口
		stream, err := chatModel.Stream(r.Context(), messages)
		if err != nil {
			sendErrorEvent(w, flusher, fmt.Sprintf("调用模型失败: %v", err))
			return
		}

		// 流式发送响应
		for {
			chunk, err := stream.Recv()
			if err != nil {
				break
			}

			event := ChatEvent{
				Type:    "text",
				Content: chunk.Content,
				Done:    false,
			}

			jsonData, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()
		}

		// 发送完成事件
		event := ChatEvent{
			Type:    "text",
			Content: "",
			Done:    true,
		}
		jsonData, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	})

	// 提供前端页面
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>A2UI 流式对话</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f5f5f5;
            height: 100vh;
            display: flex;
            flex-direction: column;
        }

        .container {
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            flex: 1;
            display: flex;
            flex-direction: column;
        }

        h1 {
            text-align: center;
            color: #333;
            margin-bottom: 20px;
        }

        .chat-box {
            flex: 1;
            background: white;
            border-radius: 8px;
            padding: 20px;
            overflow-y: auto;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }

        .message {
            margin-bottom: 15px;
            padding: 10px 15px;
            border-radius: 8px;
            max-width: 80%;
            word-wrap: break-word;
        }

        .user-message {
            background: #007bff;
            color: white;
            margin-left: auto;
        }

        .ai-message {
            background: #e9ecef;
            color: #333;
        }

        .input-area {
            display: flex;
            gap: 10px;
            margin-top: 20px;
        }

        input[type="text"] {
            flex: 1;
            padding: 12px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 16px;
        }

        button {
            padding: 12px 24px;
            background: #007bff;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
        }

        button:hover {
            background: #0056b3;
        }

        button:disabled {
            background: #ccc;
            cursor: not-allowed;
        }

        .typing-indicator {
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 2px solid #666;
            border-radius: 50%;
            border-top-color: transparent;
            animation: spin 1s linear infinite;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>A2UI 流式对话</h1>

        <div class="chat-box" id="chatBox">
            <!-- 消息将在这里显示 -->
        </div>

        <div class="input-area">
            <input type="text" id="userInput" placeholder="输入消息..." onkeypress="handleKeyPress(event)">
            <button id="sendBtn" onclick="sendMessage()">发送</button>
        </div>
    </div>

    <script>
        const chatBox = document.getElementById('chatBox');
        const userInput = document.getElementById('userInput');
        const sendBtn = document.getElementById('sendBtn');

        let currentMessage = null;
        let eventSource = null;

        function addMessage(text, isUser) {
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ' + (isUser ? 'user-message' : 'ai-message');
            messageDiv.textContent = text;
            chatBox.appendChild(messageDiv);
            chatBox.scrollTop = chatBox.scrollHeight;
            return messageDiv;
        }

        function sendMessage() {
            const message = userInput.value.trim();
            if (!message) return;

            // 添加用户消息
            addMessage(message, true);
            userInput.value = '';

            // 禁用发送按钮
            sendBtn.disabled = true;

            // 创建 AI 消息容器
            currentMessage = addMessage('', false);
            currentMessage.innerHTML = '<span class="typing-indicator"></span>';

            // 建立 SSE 连接
            eventSource = new EventSource('/api/chat?message=' + encodeURIComponent(message));

            let fullText = '';

            eventSource.onmessage = function(event) {
                const data = JSON.parse(event.data);

                if (data.type === 'text') {
                    if (data.done) {
                        // 流结束
                        currentMessage.textContent = fullText;
                        sendBtn.disabled = false;
                        eventSource.close();
                    } else {
                        // 追加文本
                        fullText += data.content;
                        currentMessage.textContent = fullText;
                    }
                } else if (data.type === 'error') {
                    currentMessage.textContent = '错误: ' + data.content;
                    currentMessage.style.color = 'red';
                    sendBtn.disabled = false;
                    eventSource.close();
                }
            };

            eventSource.onerror = function() {
                currentMessage.textContent = '连接中断';
                currentMessage.style.color = 'red';
                sendBtn.disabled = false;
                eventSource.close();
            };
        }

        function handleKeyPress(event) {
            if (event.key === 'Enter') {
                sendMessage();
            }
        }

        // 自动聚焦输入框
        userInput.focus();
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	})

	fmt.Println("聊天服务启动在 http://localhost:8080")
	fmt.Println("访问 http://localhost:8080 开始对话")
	http.ListenAndServe(":8080", nil)
}

// sendErrorEvent 发送错误事件
func sendErrorEvent(w http.ResponseWriter, flusher http.Flusher, message string) {
	event := ChatEvent{
		Type:    "error",
		Content: message,
		Done:    true,
	}
	jsonData, _ := json.Marshal(event)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
}
