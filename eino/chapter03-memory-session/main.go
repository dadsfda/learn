package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

// =============================================================================
// 第 3 章：Memory 和 Session —— 持久化会话
// =============================================================================
//
// 本章学习目标：
// 1. 理解 Memory 和 Session 的核心概念
// 2. 实现基于内存的会话管理
// 3. 实现基于文件的持久化存储（JSONL 格式）
// 4. 实现跨会话的状态恢复
//
// 运行方式：
//   go run main.go memory      - 运行内存会话示例
//   go run main.go persist     - 运行持久化会话示例
//   go run main.go interactive - 运行交互式对话（支持会话恢复）
// =============================================================================

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "memory":
			memorySessionExample()
		case "persist":
			persistentSessionExample()
		case "interactive":
			interactiveSession()
		default:
			printUsage()
		}
	} else {
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Eino Memory 和 Session 示例程序")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  go run main.go memory      - 运行内存会话示例")
	fmt.Println("  go run main.go persist     - 运行持久化会话示例")
	fmt.Println("  go run main.go interactive - 运行交互式对话（支持会话恢复）")
	fmt.Println("")
	fmt.Println("环境变量:")
	fmt.Println("  OPENAI_API_KEY - OpenAI API 密钥（必需）")
	fmt.Println("  SESSION_DIR    - 会话存储目录（可选，默认: ./data/sessions）")
}

// =============================================================================
// 核心概念：Session（会话）
// =============================================================================
//
// Session 代表一次完整的对话。它包含：
// - ID：唯一标识符
// - CreatedAt：创建时间
// - Messages：消息列表
//
// Session 是业务层概念，不是 Eino 框架的核心组件。
// 框架只负责处理消息列表，业务层负责存储和管理。
// =============================================================================

// Session 表示一个对话会话
type Session struct {
	ID        string             `json:"id"`         // 会话唯一标识
	CreatedAt time.Time          `json:"created_at"` // 创建时间
	Title     string             `json:"title"`      // 会话标题（从第一条用户消息生成）
	messages  []*schema.Message  // 对话消息列表
	mu        sync.Mutex         // 并发安全锁
	filePath  string             // 持久化文件路径（空表示仅内存）
}

// NewSession 创建一个新的会话
func NewSession(id string) *Session {
	return &Session{
		ID:        id,
		CreatedAt: time.Now(),
		messages:  make([]*schema.Message, 0),
	}
}

// Append 添加一条消息到会话
func (s *Session) Append(msg *schema.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, msg)

	// 如果是第一条用户消息，自动生成标题
	if s.Title == "" && msg.Role == schema.User {
		title := msg.Content
		if len([]rune(title)) > 50 {
			title = string([]rune(title)[:50]) + "..."
		}
		s.Title = title
	}

	// 如果设置了文件路径，同时持久化到文件
	if s.filePath != "" {
		return s.appendToFile(msg)
	}

	return nil
}

// GetMessages 获取所有消息的副本
func (s *Session) GetMessages() []*schema.Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]*schema.Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// appendToFile 将消息追加到 JSONL 文件
// JSONL 格式：每行一个 JSON 对象，便于追加和读取
func (s *Session) appendToFile(msg *schema.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	f, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

// =============================================================================
// 核心概念：Store（存储）
// =============================================================================
//
// Store 负责管理多个 Session，提供：
// - 创建/获取会话
// - 列出所有会话
// - 删除会话
//
// Store 支持多种存储后端：
// - 内存存储：适合开发和测试
// - 文件存储：适合单机部署
// - 数据库存储：适合生产环境（MySQL、PostgreSQL、MongoDB）
// - Redis 存储：适合分布式部署
// =============================================================================

// SessionMeta 会话元数据，用于列表展示
type SessionMeta struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// Store 会话存储管理器
type Store struct {
	dir   string                // 存储目录
	cache map[string]*Session   // 内存缓存
	mu    sync.Mutex            // 并发安全锁
}

// NewStore 创建一个新的存储管理器
func NewStore(dir string) (*Store, error) {
	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}

	return &Store{
		dir:   dir,
		cache: make(map[string]*Session),
	}, nil
}

// GetOrCreate 获取或创建会话
func (s *Store) GetOrCreate(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 先从缓存中查找
	if sess, ok := s.cache[id]; ok {
		return sess, nil
	}

	// 检查文件是否存在
	filePath := filepath.Join(s.dir, id+".jsonl")
	var sess *Session
	var err error

	if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
		// 创建新会话
		sess, err = s.createSession(id, filePath)
	} else {
		// 从文件加载会话
		sess, err = s.loadSession(id, filePath)
	}

	if err != nil {
		return nil, err
	}

	s.cache[id] = sess
	return sess, nil
}

// List 列出所有会话的元数据
func (s *Store) List() ([]SessionMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	var metas []SessionMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}

		id := strings.TrimSuffix(e.Name(), ".jsonl")

		// 优先从缓存获取
		if sess, ok := s.cache[id]; ok {
			metas = append(metas, SessionMeta{
				ID:        sess.ID,
				Title:     sess.Title,
				CreatedAt: sess.CreatedAt,
			})
			continue
		}

		// 从文件加载
		sess, loadErr := s.loadSession(id, filepath.Join(s.dir, e.Name()))
		if loadErr != nil {
			continue // 跳过损坏的文件
		}

		metas = append(metas, SessionMeta{
			ID:        sess.ID,
			Title:     sess.Title,
			CreatedAt: sess.CreatedAt,
		})
	}

	return metas, nil
}

// Delete 删除会话
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := filepath.Join(s.dir, id+".jsonl")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除文件失败: %w", err)
	}

	delete(s.cache, id)
	return nil
}

// sessionHeader JSONL 文件的第一行，存储会话元数据
type sessionHeader struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

// createSession 创建新会话并写入文件头
func (s *Store) createSession(id, filePath string) (*Session, error) {
	header := sessionHeader{
		Type:      "session",
		ID:        id,
		CreatedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(header)
	if err != nil {
		return nil, fmt.Errorf("序列化会话头失败: %w", err)
	}

	// 写入文件头（第一行）
	if err := os.WriteFile(filePath, append(data, '\n'), 0644); err != nil {
		return nil, fmt.Errorf("写入会话头失败: %w", err)
	}

	return &Session{
		ID:        header.ID,
		CreatedAt: header.CreatedAt,
		filePath:  filePath,
		messages:  make([]*schema.Message, 0),
	}, nil
}

// loadSession 从 JSONL 文件加载会话
func (s *Store) loadSession(id, filePath string) (*Session, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开会话文件失败: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// 读取第一行：会话头
	if !scanner.Scan() {
		return nil, fmt.Errorf("空的会话文件: %s", filePath)
	}

	var header sessionHeader
	if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
		return nil, fmt.Errorf("解析会话头失败: %w", err)
	}

	sess := &Session{
		ID:        header.ID,
		CreatedAt: header.CreatedAt,
		filePath:  filePath,
		messages:  make([]*schema.Message, 0),
	}

	// 读取后续行：消息记录
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg schema.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue // 跳过损坏的行
		}
		sess.messages = append(sess.messages, &msg)

		// 更新标题
		if sess.Title == "" && msg.Role == schema.User {
			title := msg.Content
			if len([]rune(title)) > 50 {
				title = string([]rune(title)[:50]) + "..."
			}
			sess.Title = title
		}
	}

	return sess, scanner.Err()
}

// =============================================================================
// 示例 1：内存会话管理
// =============================================================================
//
// 这个示例演示如何使用内存中的消息列表管理多轮对话。
// 注意：这种方式在程序退出后会丢失所有对话历史。
// =============================================================================

func memorySessionExample() {
	fmt.Println("=== 示例 1：内存会话管理 ===")
	fmt.Println("")
	fmt.Println("这个示例演示基于内存的会话管理。")
	fmt.Println("对话历史仅保存在内存中，程序退出后会丢失。")
	fmt.Println("")

	// 创建会话
	sessionID := uuid.New().String()
	session := NewSession(sessionID)
	fmt.Printf("创建会话: %s\n", sessionID)

	// 模拟对话历史
	conversations := []struct {
		role    schema.RoleType
		content string
	}{
		{schema.User, "你好，我叫小明，请记住我的名字。"},
		{schema.Assistant, "你好小明！很高兴认识你。我会记住你的名字。"},
		{schema.User, "我是一名 Go 语言开发者。"},
		{schema.Assistant, "太好了！Go 是一门很棒的语言，特别适合构建高性能的后端服务。"},
		{schema.User, "请回顾一下我们刚才的对话。"},
	}

	// 添加消息到会话
	for _, conv := range conversations {
		msg := &schema.Message{
			Role:    conv.role,
			Content: conv.content,
		}
		if err := session.Append(msg); err != nil {
			fmt.Printf("添加消息失败: %v\n", err)
			os.Exit(1)
		}
	}

	// 显示会话信息
	fmt.Printf("\n会话 ID: %s\n", session.ID)
	fmt.Printf("会话标题: %s\n", session.Title)
	fmt.Printf("消息数量: %d\n", len(session.GetMessages()))

	// 显示所有消息
	fmt.Println("\n--- 对话历史 ---")
	for i, msg := range session.GetMessages() {
		role := "用户"
		if msg.Role == schema.Assistant {
			role = "助手"
		}
		fmt.Printf("[%d] %s: %s\n", i+1, role, msg.Content)
	}

	// 演示消息作为上下文传递给模型
	fmt.Println("\n--- 准备发送给模型的消息 ---")
	history := session.GetMessages()
	fmt.Printf("将 %d 条消息作为上下文发送给 LLM\n", len(history))
	fmt.Println("提示: LLM 可以根据完整的对话历史生成连贯的回复")
}

// =============================================================================
// 示例 2：持久化会话存储
// =============================================================================
//
// 这个示例演示如何将会话持久化到 JSONL 文件。
// JSONL 格式的优势：
// - 简单：每行一个 JSON 对象
// - 可追加：新消息直接追加到文件末尾
// - 可读：人类可以直接阅读文件内容
// - 容错：单行损坏不影响其他消息
// =============================================================================

func persistentSessionExample() {
	fmt.Println("=== 示例 2：持久化会话存储 ===")
	fmt.Println("")
	fmt.Println("这个示例演示基于 JSONL 文件的持久化会话存储。")
	fmt.Println("对话历史会保存到文件，程序退出后可以恢复。")
	fmt.Println("")

	// 获取存储目录
	sessionDir := os.Getenv("SESSION_DIR")
	if sessionDir == "" {
		sessionDir = "./data/sessions"
	}

	// 创建存储管理器
	store, err := NewStore(sessionDir)
	if err != nil {
		fmt.Printf("创建存储管理器失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("存储目录: %s\n", sessionDir)

	// 创建第一个会话
	sessionID1 := uuid.New().String()
	session1, err := store.GetOrCreate(sessionID1)
	if err != nil {
		fmt.Printf("创建会话失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n创建会话 1: %s\n", sessionID1)

	// 添加一些消息
	messages1 := []struct {
		role    schema.RoleType
		content string
	}{
		{schema.User, "什么是 Go 语言？"},
		{schema.Assistant, "Go 是 Google 开发的开源编程语言，以简洁、高效、并发支持著称。"},
		{schema.User, "Go 有什么优势？"},
		{schema.Assistant, "Go 的主要优势包括：编译速度快、内置并发支持、垃圾回收、跨平台编译等。"},
	}

	for _, msg := range messages1 {
		if err := session1.Append(&schema.Message{
			Role:    msg.role,
			Content: msg.content,
		}); err != nil {
			fmt.Printf("添加消息失败: %v\n", err)
			os.Exit(1)
		}
	}

	// 创建第二个会话
	sessionID2 := uuid.New().String()
	session2, err := store.GetOrCreate(sessionID2)
	if err != nil {
		fmt.Printf("创建会话失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("创建会话 2: %s\n", sessionID2)

	// 添加消息到第二个会话
	if err := session2.Append(&schema.Message{
		Role:    schema.User,
		Content: "帮我写一个 Hello World 程序",
	}); err != nil {
		fmt.Printf("添加消息失败: %v\n", err)
		os.Exit(1)
	}

	if err := session2.Append(&schema.Message{
		Role:    schema.Assistant,
		Content: "好的，这是一个简单的 Go Hello World 程序：\n\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}",
	}); err != nil {
		fmt.Printf("添加消息失败: %v\n", err)
		os.Exit(1)
	}

	// 列出所有会话
	fmt.Println("\n--- 所有会话列表 ---")
	sessions, err := store.List()
	if err != nil {
		fmt.Printf("列出会话失败: %v\n", err)
		os.Exit(1)
	}

	for i, meta := range sessions {
		fmt.Printf("[%d] ID: %s\n", i+1, meta.ID)
		fmt.Printf("    标题: %s\n", meta.Title)
		fmt.Printf("    创建时间: %s\n", meta.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	// 演示会话恢复
	fmt.Println("\n--- 演示会话恢复 ---")
	fmt.Printf("正在恢复会话: %s\n", sessionID1)

	restoredSession, err := store.GetOrCreate(sessionID1)
	if err != nil {
		fmt.Printf("恢复会话失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("恢复的会话标题: %s\n", restoredSession.Title)
	fmt.Printf("恢复的消息数量: %d\n", len(restoredSession.GetMessages()))

	fmt.Println("\n恢复的消息:")
	for i, msg := range restoredSession.GetMessages() {
		role := "用户"
		if msg.Role == schema.Assistant {
			role = "助手"
		}
		fmt.Printf("  [%d] %s: %s\n", i+1, role, msg.Content)
	}

	// 显示文件位置
	fmt.Printf("\n会话文件保存在: %s\n", sessionDir)
	fmt.Println("你可以直接查看 .jsonl 文件内容")
}

// =============================================================================
// 示例 3：交互式对话（支持会话恢复）
// =============================================================================
//
// 这个示例演示完整的交互式对话系统，支持：
// - 创建新会话
// - 恢复已有会话
// - 持久化对话历史
// - 跨进程恢复对话
//
// 使用方式：
//   go run main.go interactive                    # 创建新会话
//   go run main.go interactive --session <id>     # 恢复已有会话
// =============================================================================

func interactiveSession() {
	// 检查 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("错误: 请设置 OPENAI_API_KEY 环境变量")
		fmt.Println("  export OPENAI_API_KEY=\"your-api-key-here\"")
		os.Exit(1)
	}

	// 解析命令行参数
	var sessionID string
	for i, arg := range os.Args {
		if arg == "--session" && i+1 < len(os.Args) {
			sessionID = os.Args[i+1]
			break
		}
	}

	ctx := context.Background()
	_ = ctx // 在实际应用中，ctx 会传递给 ChatModel.Generate() 等方法

	// 获取存储目录
	sessionDir := os.Getenv("SESSION_DIR")
	if sessionDir == "" {
		sessionDir = "./data/sessions"
	}

	// 创建存储管理器
	store, err := NewStore(sessionDir)
	if err != nil {
		fmt.Printf("创建存储管理器失败: %v\n", err)
		os.Exit(1)
	}

	// 获取或创建会话
	if sessionID == "" {
		sessionID = uuid.New().String()
		fmt.Printf("创建新会话: %s\n", sessionID)
	} else {
		fmt.Printf("恢复会话: %s\n", sessionID)
	}

	session, err := store.GetOrCreate(sessionID)
	if err != nil {
		fmt.Printf("获取会话失败: %v\n", err)
		os.Exit(1)
	}

	if session.Title != "" {
		fmt.Printf("会话标题: %s\n", session.Title)
	}

	// 注意：这里为了简化示例，我们不实际调用 LLM API
	// 在实际应用中，你需要创建 ChatModel 并调用 Generate 方法
	//
	// 示例代码（需要取消注释并配置 API Key）：
	//
	// chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
	//     Model:  "gpt-4o",
	//     APIKey: apiKey,
	// })
	// if err != nil {
	//     fmt.Printf("创建 ChatModel 失败: %v\n", err)
	//     os.Exit(1)
	// }

	fmt.Println("\n=== 交互式对话系统 ===")
	fmt.Println("命令:")
	fmt.Println("  'quit'   - 退出程序")
	fmt.Println("  'clear'  - 清空当前会话历史")
	fmt.Println("  'list'   - 列出所有会话")
	fmt.Println("  'switch' - 切换到其他会话")
	fmt.Println("  'info'   - 显示当前会话信息")
	fmt.Println("")
	fmt.Printf("会话 ID: %s\n", sessionID)
	fmt.Println("")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("你: ")
		if !scanner.Scan() {
			break
		}

	 userInput := strings.TrimSpace(scanner.Text())

		// 处理命令
		switch userInput {
		case "quit":
			fmt.Printf("\n会话已保存: %s\n", sessionID)
			fmt.Printf("恢复会话: go run main.go interactive --session %s\n", sessionID)
			fmt.Println("再见！")
			return

		case "clear":
			// 创建新会话替换当前会话
			sessionID = uuid.New().String()
			session, err = store.GetOrCreate(sessionID)
			if err != nil {
				fmt.Printf("创建新会话失败: %v\n", err)
				continue
			}
			fmt.Printf("✓ 已创建新会话: %s\n", sessionID)
			continue

		case "list":
			sessions, err := store.List()
			if err != nil {
				fmt.Printf("列出会话失败: %v\n", err)
				continue
			}
			fmt.Println("\n--- 所有会话 ---")
			for i, meta := range sessions {
				marker := " "
				if meta.ID == sessionID {
					marker = "*"
				}
				fmt.Printf("%s [%d] %s - %s\n", marker, i+1, meta.ID[:8]+"...", meta.Title)
			}
			fmt.Println("")
			continue

		case "switch":
			sessions, err := store.List()
			if err != nil {
				fmt.Printf("列出会话失败: %v\n", err)
				continue
			}
			if len(sessions) == 0 {
				fmt.Println("没有其他会话")
				continue
			}
			fmt.Println("\n--- 选择会话 ---")
			for i, meta := range sessions {
				fmt.Printf("[%d] %s - %s\n", i+1, meta.ID[:8]+"...", meta.Title)
			}
			fmt.Print("输入序号: ")
			if !scanner.Scan() {
				break
			}
			var idx int
			if _, err := fmt.Sscanf(scanner.Text(), "%d", &idx); err != nil || idx < 1 || idx > len(sessions) {
				fmt.Println("无效的序号")
				continue
			}
			sessionID = sessions[idx-1].ID
			session, err = store.GetOrCreate(sessionID)
			if err != nil {
				fmt.Printf("切换会话失败: %v\n", err)
				continue
			}
			fmt.Printf("✓ 已切换到会话: %s\n", sessionID)
			fmt.Printf("  标题: %s\n", session.Title)
			fmt.Printf("  消息数: %d\n", len(session.GetMessages()))
			continue

		case "info":
			fmt.Printf("\n--- 当前会话信息 ---\n")
			fmt.Printf("ID: %s\n", session.ID)
			fmt.Printf("标题: %s\n", session.Title)
			fmt.Printf("创建时间: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("消息数量: %d\n", len(session.GetMessages()))
			fmt.Println("")
			continue

		case "":
			continue
		}

		// 保存用户消息
		userMsg := &schema.Message{
			Role:    schema.User,
			Content: userInput,
		}
		if err := session.Append(userMsg); err != nil {
			fmt.Printf("保存用户消息失败: %v\n", err)
			continue
		}

		// 获取对话历史
		history := session.GetMessages()

		// 在实际应用中，这里会调用 LLM API：
		//
		// resp, err := chatModel.Generate(ctx, history)
		// if err != nil {
		//     fmt.Printf("调用模型失败: %v\n", err)
		//     continue
		// }
		// assistantContent := resp.Content

		// 模拟 AI 回复（实际应用中替换为上面的代码）
		assistantContent := fmt.Sprintf(
			"[模拟回复] 我收到了你的消息: %s\n"+
				"（这是示例程序，实际应用需要配置 OPENAI_API_KEY 来调用 LLM API）\n"+
				"当前对话历史包含 %d 条消息",
			userInput, len(history))

		// 保存助手回复
		assistantMsg := &schema.Message{
			Role:    schema.Assistant,
			Content: assistantContent,
		}
		if err := session.Append(assistantMsg); err != nil {
			fmt.Printf("保存助手消息失败: %v\n", err)
			continue
		}

		// 显示回复
		fmt.Printf("\nAI: %s\n\n", assistantContent)
	}
}
