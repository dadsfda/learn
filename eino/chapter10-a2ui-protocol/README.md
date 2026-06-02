# 第 10 章：A2UI Protocol —— 流式 UI 组件

## 学习目标

通过本章学习，你将掌握：

1. **A2UI 协议**：理解 Agent 到 UI 的流式通信协议
2. **HTTP SSE 服务**：实现 Server-Sent Events 流式推送
3. **Runner HTTP 封装**：将 Eino Runner 转换为 HTTP 服务
4. **前端集成**：在 Web 页面中接收和渲染流式数据
5. **实时 UI 更新**：实现 AI 响应的实时流式显示

## 前置知识

- Go 语言 HTTP 服务开发
- Eino ChatModel 和 Runner 基础（第 1-2 章）
- HTML/JavaScript 基础
- 事件驱动编程概念

## 核心概念

### 1. A2UI 协议简介

A2UI（Agent-to-UI）协议是一种用于 AI Agent 与前端 UI 之间流式通信的协议。它解决了以下问题：

**传统方式的问题**：
- 用户发送请求后需要等待完整响应
- 无法实时看到 AI 的生成过程
- 前端无法知道 Agent 的中间状态

**A2UI 的解决方案**：
- 使用 HTTP SSE（Server-Sent Events）实现单向流式推送
- Agent 可以逐步发送生成的内容
- 前端实时接收并渲染更新

### 2. HTTP SSE（Server-Sent Events）

SSE 是一种服务器向客户端推送数据的技术：

```
客户端                    服务器
  |                         |
  |--- HTTP 请求 --------->|
  |                         |
  |<-- SSE 事件流 ---------|
  |    (text/event-stream)  |
  |                         |
  |<-- 事件 1 --------------|
  |<-- 事件 2 --------------|
  |<-- 事件 3 --------------|
  |         ...             |
  |<-- 事件结束 ------------|
```

**SSE 的特点**：
- 单向通信：服务器 -> 客户端
- 基于 HTTP 协议
- 自动重连机制
- 文本格式传输

### 3. A2UI 事件类型

A2UI 协议定义了多种事件类型：

```go
// 文本内容事件
type TextEvent struct {
    Type    string `json:"type"`    // "text"
    Content string `json:"content"` // 文本内容
    Done    bool   `json:"done"`    // 是否结束
}

// 工具调用事件
type ToolCallEvent struct {
    Type     string `json:"type"`     // "tool_call"
    Name     string `json:"name"`     // 工具名称
    Args     string `json:"args"`     // 工具参数
    Status   string `json:"status"`   // 状态：pending/running/done
}

// 状态更新事件
type StatusEvent struct {
    Type    string `json:"type"`    // "status"
    Status  string `json:"status"`  // 状态信息
    Message string `json:"message"` // 描述信息
}

// 错误事件
type ErrorEvent struct {
    Type    string `json:"type"`    // "error"
    Code    int    `json:"code"`    // 错误码
    Message string `json:"message"` // 错误信息
}
```

### 4. Runner HTTP 服务封装

Eino 的 Runner 可以封装为 HTTP 服务：

```go
// RunnerHTTPHandler 将 Runner 封装为 HTTP 处理器
type RunnerHTTPHandler struct {
    runner *runner.Runner
}

// ServeHTTP 处理 HTTP 请求
func (h *RunnerHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. 设置 SSE 响应头
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    // 2. 获取用户输入
    userInput := r.URL.Query().Get("message")

    // 3. 调用 Runner 的流式接口
    stream, err := h.runner.Stream(r.Context(), userInput)
    if err != nil {
        // 发送错误事件
        sendErrorEvent(w, err)
        return
    }

    // 4. 流式发送事件
    flusher := w.(http.Flusher)
    for {
        chunk, err := stream.Recv()
        if err != nil {
            break
        }
        // 发送文本事件
        sendTextEvent(w, chunk.Content, false)
        flusher.Flush()
    }

    // 5. 发送结束事件
    sendTextEvent(w, "", true)
    flusher.Flush()
}
```

### 5. 前端集成

前端使用 JavaScript 接收 SSE 事件：

```javascript
// 创建 EventSource 连接
const eventSource = new EventSource('/api/chat?message=你好');

// 监听消息事件
eventSource.onmessage = function(event) {
    const data = JSON.parse(event.data);

    switch(data.type) {
        case 'text':
            appendText(data.content);
            if (data.done) {
                showComplete();
            }
            break;
        case 'tool_call':
            showToolCall(data.name, data.status);
            break;
        case 'status':
            showStatus(data.message);
            break;
        case 'error':
            showError(data.message);
            break;
    }
};

// 监听连接关闭
eventSource.onerror = function() {
    console.log('连接关闭');
    eventSource.close();
};
```

## 代码示例

### 示例 1：基础 HTTP SSE 服务

```go
package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	// 注册 SSE 处理器
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
			fmt.Fprintf(w, "data: 消息 %d\n\n", i)
			flusher.Flush()
			time.Sleep(time.Second)
		}

		// 发送结束事件
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	})

	fmt.Println("SSE 服务启动在 http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
```

### 示例 2：带 JSON 格式的 SSE 服务

```go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SSEEvent 表示 SSE 事件结构
type SSEEvent struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

func main() {
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
		texts := []string{"你", "好", "，", "我", "是", "AI", "助", "手", "！"}
		for _, text := range texts {
			event := SSEEvent{
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
		event := SSEEvent{
			Type:    "text",
			Content: "",
			Done:    true,
		}
		jsonData, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	})

	fmt.Println("服务启动在 http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
```

### 示例 3：集成 Eino Runner 的 HTTP 服务

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// ChatRequest 表示聊天请求
type ChatRequest struct {
	Message string `json:"message"`
}

// ChatEvent 表示聊天事件
type ChatEvent struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

func main() {
	ctx := context.Background()

	// 创建 ChatModel
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("错误: 请设置 OPENAI_API_KEY 环境变量")
		os.Exit(1)
	}

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
			sendError(w, flusher, "消息不能为空")
			return
		}

		// 构建消息列表
		messages := []*model.Message{
			model.SystemMessage("你是一个 helpful 的助手。"),
			model.UserMessage(message),
		}

		// 使用流式接口
		stream, err := chatModel.Stream(r.Context(), messages)
		if err != nil {
			sendError(w, flusher, fmt.Sprintf("调用模型失败: %v", err))
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

	// 提供静态文件服务
	http.Handle("/", http.FileServer(http.Dir("./static")))

	fmt.Println("服务启动在 http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func sendError(w http.ResponseWriter, flusher http.Flusher, message string) {
	event := ChatEvent{
		Type:    "error",
		Content: message,
		Done:    true,
	}
	jsonData, _ := json.Marshal(event)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
}
```

### 示例 4：完整前端示例

创建 `static/index.html` 文件：

```html
<!DOCTYPE html>
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
            messageDiv.className = `message ${isUser ? 'user-message' : 'ai-message'}`;
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
            eventSource = new EventSource(`/api/chat?message=${encodeURIComponent(message)}`);

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
                    currentMessage.textContent = `错误: ${data.content}`;
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
</html>
```

## 运行步骤

### 1. 环境准备

```bash
# 确保 Go 版本 >= 1.18
go version

# 设置 OpenAI API Key
export OPENAI_API_KEY="your-api-key-here"
```

### 2. 初始化项目

```bash
# 进入章节目录
cd chapter10-a2ui-protocol

# 初始化 Go 模块
go mod init chapter10

# 安装依赖
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino-ext/components/model/openai
```

### 3. 创建项目结构

```
chapter10-a2ui-protocol/
├── main.go           # 后端服务代码
├── static/
│   └── index.html    # 前端页面
├── go.mod
└── go.sum
```

### 4. 创建静态文件目录

```bash
mkdir static
```

### 5. 创建前端文件

将上面的 HTML 代码保存为 `static/index.html`。

### 6. 运行程序

```bash
# 运行服务
go run main.go

# 输出：
# 服务启动在 http://localhost:8080
```

### 7. 访问应用

在浏览器中打开 http://localhost:8080，即可开始流式对话。

## 常见问题

### Q1: 为什么 SSE 连接立即关闭？

**原因**：可能是响应头设置不正确。

**解决**：
```go
// 确保设置以下响应头
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Connection", "keep-alive")
```

### Q2: 如何处理客户端断开连接？

**解决**：使用 `r.Context()` 监听连接关闭：

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // 使用 context 监听连接关闭
    go func() {
        <-ctx.Done()
        fmt.Println("客户端断开连接")
        // 清理资源
    }()

    // ... 处理逻辑
}
```

### Q3: 如何实现多轮对话？

**解决**：维护会话状态：

```go
// 使用 map 存储会话历史
var sessions = make(map[string][]*model.Message)

func handler(w http.ResponseWriter, r *http.Request) {
    sessionID := r.URL.Query().Get("session_id")
    message := r.URL.Query().Get("message")

    // 获取或创建会话历史
    history, exists := sessions[sessionID]
    if !exists {
        history = []*model.Message{
            model.SystemMessage("你是一个 helpful 的助手。"),
        }
    }

    // 添加用户消息
    history = append(history, model.UserMessage(message))

    // ... 调用模型

    // 更新会话历史
    sessions[sessionID] = history
}
```

### Q4: 如何部署到生产环境？

**建议**：
1. 使用反向代理（如 Nginx）处理 SSE 连接
2. 配置适当的超时时间
3. 使用连接池管理数据库连接
4. 添加日志和监控

**Nginx 配置示例**：
```nginx
location /api/chat {
    proxy_pass http://localhost:8080;
    proxy_http_version 1.1;
    proxy_set_header Connection '';
    proxy_buffering off;
    proxy_cache off;
    chunked_transfer_encoding off;
}
```

### Q5: 如何优化性能？

**建议**：
1. 使用 `sync.Pool` 复用对象
2. 实现连接池
3. 添加缓存层
4. 使用 goroutine 池处理并发请求

```go
// 使用 goroutine 池
var pool = make(chan struct{}, 100)

func handler(w http.ResponseWriter, r *http.Request) {
    pool <- struct{}{}        // 获取令牌
    defer func() { <-pool }() // 释放令牌

    // ... 处理逻辑
}
```

## 练习题

### 练习 1：基础 SSE 服务

创建一个简单的 SSE 服务，要求：
1. 每秒发送一个时间戳事件
2. 发送 10 个事件后结束
3. 前端页面显示接收到的时间

### 练习 2：流式文本生成

实现一个流式文本生成服务，要求：
1. 模拟 AI 逐字生成文本
2. 前端实时显示生成过程
3. 添加打字机效果

### 练习 3：多事件类型

扩展 A2UI 协议，支持多种事件类型：
1. 文本事件（text）
2. 状态事件（status）
3. 进度事件（progress）
4. 错误事件（error）

### 练习 4：会话管理

实现完整的会话管理功能：
1. 支持多个会话
2. 会话历史持久化
3. 会话超时清理
4. 会话列表查询

### 练习 5：前端优化

优化前端界面，添加以下功能：
1. Markdown 渲染
2. 代码高亮
3. 消息复制
4. 响应式设计

## 进阶学习

### 1. WebSocket 双向通信

当需要双向通信时，可以考虑使用 WebSocket：

```go
import "github.com/gorilla/websocket"

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer conn.Close()

    for {
        // 读取消息
        messageType, message, err := conn.ReadMessage()
        if err != nil {
            break
        }

        // 处理消息并发送响应
        response := processMessage(message)
        conn.WriteMessage(messageType, response)
    }
}
```

### 2. GraphQL 订阅

对于更复杂的数据需求，可以考虑 GraphQL 订阅：

```graphql
subscription {
  chatStream(message: "你好") {
    content
    done
  }
}
```

### 3. gRPC 流式调用

对于高性能场景，可以使用 gRPC 流式调用：

```protobuf
service ChatService {
  rpc ChatStream (ChatRequest) returns (stream ChatResponse);
}
```

## 下一步学习

完成本章后，建议继续学习：

- **第 11 章**：TurnLoop —— 实现完整的对话循环
- **第 12 章**：多模态处理 —— 支持图片、音频等多媒体

## 参考资料

- [Eino 官方文档](https://www.cloudwego.io/docs/eino/)
- [MDN - Server-Sent Events](https://developer.mozilla.org/zh-CN/docs/Web/API/Server-sent_events)
- [Go HTTP 服务开发](https://go.dev/doc/articles/wiki/)
- [EventSource API](https://developer.mozilla.org/zh-CN/docs/Web/API/EventSource)

---

**下一章**：[第 11 章：TurnLoop](../chapter11-turnloop/README.md)
