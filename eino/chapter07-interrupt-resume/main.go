// ============================================================================
// 第 7 章：Interrupt/Resume —— 人在环中（Human-in-the-Loop）
// ============================================================================
//
// 本示例演示 Eino 框架的中断/恢复机制，包括：
//   demo        - 基础中断/恢复概念演示（模拟，不需要 API Key）
//   checkpoint  - 检查点保存和恢复演示（模拟，不需要 API Key）
//   approval    - 审批流程演示（模拟，不需要 API Key）
//   review      - 审阅编辑演示（模拟，不需要 API Key）
//   interactive - 交互式 Agent 演示（模拟，不需要 API Key，需要终端交互）
//   agent       - 真实 ADK Agent 审批示例（需要 API Key）
//
// 由于 Eino 的 ADK 中断/恢复需要完整的 Agent 运行环境（API Key 等），
// 本示例采用"先理解原理，再看实际用法"的方式：
//
//   Part 1: 用纯 Go 模拟中断/恢复的核心流程（不需要 API Key）
//   Part 2: 展示如何在真实 Eino Agent 中使用中断/恢复（需要 API Key）
//
// 运行方式：
//   go run main.go demo
//   go run main.go checkpoint
//   go run main.go approval
//   go run main.go review
//   go run main.go interactive
//   go run main.go agent
//
// ============================================================================

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// 入口函数
// ============================================================================

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "demo":
			demoBasicInterrupt()
		case "checkpoint":
			demoCheckpoint()
		case "approval":
			demoApprovalFlow()
		case "review":
			demoReviewEdit()
		case "interactive":
			demoInteractiveAgent()
		case "agent":
			demoRealAgent()
		default:
			printUsage()
		}
	} else {
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Eino Interrupt/Resume 示例程序")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  go run main.go demo        - 基础中断/恢复概念演示")
	fmt.Println("  go run main.go checkpoint  - 检查点保存和恢复演示")
	fmt.Println("  go run main.go approval    - 审批流程演示")
	fmt.Println("  go run main.go review      - 审阅编辑演示")
	fmt.Println("  go run main.go interactive - 交互式 Agent 演示")
	fmt.Println("  go run main.go agent       - 真实 ADK Agent 审批示例")
	fmt.Println("")
	fmt.Println("环境变量:")
	fmt.Println("  OPENAI_API_KEY - OpenAI API 密钥（仅 agent 示例需要）")
}

// ============================================================================
// Part 1: 模拟中断/恢复系统（帮助理解原理）
// ============================================================================
//
// 这部分用纯 Go 代码模拟 Eino 的中断/恢复机制。
// 不需要任何外部依赖，帮助你理解核心思想。
//
// ============================================================================

// --------------------------------------------------------------------------
// 核心类型定义
// --------------------------------------------------------------------------

// InterruptInfo 中断信息
// 当 Agent 或工具需要人工输入时，会生成这个结构
// 在真实 Eino 中，对应 compose.InterruptInfo 或 adk.InterruptInfo
type InterruptInfo struct {
	ID      string         `json:"id"`      // 中断点的唯一标识
	Message string         `json:"message"` // 展示给用户的消息
	State   map[string]any `json:"state"`   // 中断时保存的状态数据
}

// InterruptError 中断错误
// 在 Go 中，中断通过 error 类型传播，但这不是真正的"错误"
// 框架通过类型断言区分中断和真正的错误
type InterruptError struct {
	Info *InterruptInfo
}

func (e *InterruptError) Error() string {
	return fmt.Sprintf("interrupted: %s", e.Info.Message)
}

// IsInterrupt 检查一个 error 是否是中断
// 这是 Eino 框架中区分中断和错误的标准方式
func IsInterrupt(err error) bool {
	_, ok := err.(*InterruptError)
	return ok
}

// CheckPoint 检查点数据
// 当 Agent 中断时，框架会自动保存当前状态到检查点
// 恢复时从检查点加载状态，实现"断点续传"
type CheckPoint struct {
	ID        string         `json:"id"`         // 检查点 ID
	State     map[string]any `json:"state"`      // 保存的状态
	Timestamp time.Time      `json:"timestamp"`  // 保存时间
}

// CheckPointStore 检查点存储接口
// 在真实 Eino 中，对应 adk.CheckPointStore 接口
// 可以实现内存存储、Redis 存储、数据库存储等
type CheckPointStore interface {
	Save(ctx context.Context, cp *CheckPoint) error
	Load(ctx context.Context, id string) (*CheckPoint, error)
	Delete(ctx context.Context, id string) error
}

// --------------------------------------------------------------------------
// 示例 1：基础中断/恢复概念
// --------------------------------------------------------------------------

// 示例 1：基础中断/恢复概念
// 这个示例用最简单的方式展示中断/恢复的核心流程：
//   1. Agent 执行任务
//   2. 需要人工确认时触发中断
//   3. 人工确认后恢复执行
func demoBasicInterrupt() {
	fmt.Println("=== 示例 1：基础中断/恢复概念 ===")
	fmt.Println("演示中断/恢复的核心流程，不需要 API Key")
	fmt.Println()

	ctx := context.Background()

	// ---- 第一次运行：触发中断 ----
	fmt.Println("[Step 1] Agent 开始处理任务: '帮我订一张去北京的机票'")
	fmt.Println("[Step 2] Agent 处理中...")
	fmt.Println("[Step 3] Agent 发现需要人工确认，触发中断")

	// 模拟 Agent 执行，触发中断
	_, err := simpleAgent(ctx, "帮我订一张去北京的机票", nil)
	if err != nil {
		if ie, ok := err.(*InterruptError); ok {
			fmt.Println()
			fmt.Printf("[系统] Agent 暂停！\n")
			fmt.Printf("[系统] 中断 ID: %s\n", ie.Info.ID)
			fmt.Printf("[系统] 提示信息: %s\n", ie.Info.Message)
			fmt.Printf("[系统] 等待人工输入...\n")
		}
	}

	fmt.Println()
	fmt.Println("--------------------------------------------------")
	fmt.Println()

	// ---- 模拟人工确认后恢复 ----
	fmt.Println("[人工] 用户确认: 'approved'")
	fmt.Println("[Step 4] 收到恢复信号，继续执行")

	result, err := simpleAgent(ctx, "帮我订一张去北京的机票", "approved")
	if err != nil {
		fmt.Printf("[错误] %v\n", err)
		return
	}

	fmt.Printf("[Step 5] %s\n", result)
	fmt.Println()
	fmt.Println("=== 核心流程总结 ===")
	fmt.Println("1. Agent 执行任务")
	fmt.Println("2. 需要人工输入时触发 Interrupt")
	fmt.Println("3. 框架捕获中断，保存状态")
	fmt.Println("4. 人工提供输入")
	fmt.Println("5. Agent 从断点恢复执行")
}

// simpleAgent 模拟一个简单的 Agent
// 参数：
//   - ctx: 上下文
//   - input: 用户输入
//   - resumeData: 恢复数据（nil 表示首次运行）
//
// 返回：
//   - 结果字符串
//   - 错误（可能是中断错误）
func simpleAgent(ctx context.Context, input string, resumeData any) (string, error) {
	// 如果有恢复数据，说明是从中断点恢复的
	if resumeData != nil {
		return fmt.Sprintf("任务完成！输入: '%s', 人工确认: %v", input, resumeData), nil
	}

	// 首次运行：处理到一半触发中断
	// 在真实 Eino 中，这对应 compose.StatefulInterrupt() 或 adk.Interrupt()
	return "", &InterruptError{
		Info: &InterruptInfo{
			ID:      "confirm-001",
			Message: "请确认是否继续执行订票操作",
			State: map[string]any{
				"input":      input,
				"step":       2,
				"total_step": 5,
			},
		},
	}
}

// --------------------------------------------------------------------------
// 示例 2：检查点保存和恢复
// --------------------------------------------------------------------------

// 示例 2：检查点保存和恢复
// 演示检查点（Checkpoint）的保存和恢复机制
// 检查点是中断/恢复的基础，它保存了 Agent 执行的中间状态
func demoCheckpoint() {
	fmt.Println("=== 示例 2：检查点保存和恢复 ===")
	fmt.Println("演示检查点的保存和恢复机制")
	fmt.Println()

	// 创建内存检查点存储
	store := NewMemoryCheckPointStore()
	ctx := context.Background()

	// ---- 模拟 Agent 执行并保存检查点 ----
	fmt.Println("[Phase 1] Agent 开始执行多步骤任务")
	fmt.Println()

	// 创建一个任务状态
	taskState := map[string]any{
		"task_name":   "数据处理流水线",
		"current_step": 1,
		"total_steps":  4,
		"results":      []string{},
		"status":       "running",
	}

	// 步骤 1：数据收集
	fmt.Printf("[Step 1/%d] 收集数据...\n", 4)
	taskState["results"] = append(taskState["results"].([]string), "数据收集完成")
	taskState["current_step"] = 2

	// 步骤 2：数据验证（需要人工确认）
	fmt.Printf("[Step 2/%d] 验证数据... 需要人工确认\n", 4)
	taskState["results"] = append(taskState["results"].([]string), "数据验证完成")
	taskState["status"] = "waiting_approval"
	taskState["current_step"] = 3

	// 保存检查点
	cp := &CheckPoint{
		ID:        "pipeline-001",
		State:     taskState,
		Timestamp: time.Now(),
	}
	if err := store.Save(ctx, cp); err != nil {
		fmt.Printf("[错误] 保存检查点失败: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("[系统] Agent 暂停，等待人工确认")
	fmt.Println("[系统] 检查点已保存，可以安全地关闭程序")
	fmt.Println()

	// ---- 模拟人工确认后恢复 ----
	fmt.Println("--------------------------------------------------")
	fmt.Println()
	fmt.Println("[Phase 2] 人工确认后恢复执行")
	fmt.Println("[人工] 用户确认: 'approved'")
	fmt.Println()

	// 加载检查点
	loaded, err := store.Load(ctx, "pipeline-001")
	if err != nil {
		fmt.Printf("[错误] 加载检查点失败: %v\n", err)
		return
	}

	// 从检查点恢复状态
	fmt.Printf("[Agent] 从步骤 %d/%d 恢复\n",
		loaded.State["current_step"], loaded.State["total_steps"])
	fmt.Printf("[Agent] 之前的结果: %v\n", loaded.State["results"])
	fmt.Println()

	// 继续执行剩余步骤
	fmt.Printf("[Step 3/%d] 数据转换...\n", 4)
	loaded.State["results"] = append(loaded.State["results"].([]string), "数据转换完成")
	loaded.State["current_step"] = 4

	fmt.Printf("[Step 4/%d] 数据导出...\n", 4)
	loaded.State["results"] = append(loaded.State["results"].([]string), "数据导出完成")
	loaded.State["status"] = "completed"

	// 输出最终结果
	fmt.Println()
	fmt.Println("=== 任务完成 ===")
	fmt.Printf("状态: %s\n", loaded.State["status"])
	fmt.Println("执行结果:")
	for i, r := range loaded.State["results"].([]string) {
		fmt.Printf("  %d. %s\n", i+1, r)
	}

	// 清理检查点
	store.Delete(ctx, "pipeline-001")
	fmt.Println("\n检查点已清理")
}

// MemoryCheckPointStore 内存检查点存储
// 使用 sync.Map 实现并发安全的内存存储
// 在真实 Eino 中，对应 adk.CheckPointStore 接口
type MemoryCheckPointStore struct {
	mu    sync.RWMutex
	store map[string]*CheckPoint
}

// NewMemoryCheckPointStore 创建内存检查点存储
func NewMemoryCheckPointStore() *MemoryCheckPointStore {
	return &MemoryCheckPointStore{
		store: make(map[string]*CheckPoint),
	}
}

// Save 保存检查点
func (s *MemoryCheckPointStore) Save(ctx context.Context, cp *CheckPoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[cp.ID] = cp
	fmt.Printf("[Checkpoint] 已保存: %s\n", cp.ID)
	return nil
}

// Load 加载检查点
func (s *MemoryCheckPointStore) Load(ctx context.Context, id string) (*CheckPoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp, ok := s.store[id]
	if !ok {
		return nil, fmt.Errorf("checkpoint not found: %s", id)
	}
	fmt.Printf("[Checkpoint] 已加载: %s\n", id)
	return cp, nil
}

// Delete 删除检查点
func (s *MemoryCheckPointStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.store, id)
	fmt.Printf("[Checkpoint] 已删除: %s\n", id)
	return nil
}

// --------------------------------------------------------------------------
// 示例 3：审批流程
// --------------------------------------------------------------------------

// 示例 3：审批流程
// 演示工具执行前的人工审批流程
// 这是 Human-in-the-Loop 最常见的模式
func demoApprovalFlow() {
	fmt.Println("=== 示例 3：审批流程 ===")
	fmt.Println("演示工具执行前的人工审批流程")
	fmt.Println()

	ctx := context.Background()

	// 定义审批结果类型
	// 在真实 Eino 中，对应 tool.ApprovalResult
	type ApprovalResult struct {
		Approved         bool    `json:"approved"`
		DisapproveReason *string `json:"disapprove_reason,omitempty"`
	}

	// 创建一个需要审批的工具
	// 在真实 Eino 中，使用 tool.InvokableApprovableTool 包装
	deleteTool := func(ctx context.Context, filename string, approval *ApprovalResult) (string, error) {
		// 检查是否需要审批
		if approval == nil {
			// 触发中断，请求审批
			return "", &InterruptError{
				Info: &InterruptInfo{
					ID:      "approval-delete",
					Message: fmt.Sprintf("工具 'DeleteFile' 请求执行，参数: filename='%s'", filename),
					State: map[string]any{
						"tool_name": "DeleteFile",
						"filename":  filename,
					},
				},
			}
		}

		// 处理审批结果
		if approval.Approved {
			return fmt.Sprintf("文件 '%s' 已成功删除", filename), nil
		}
		return fmt.Sprintf("操作被拒绝: %s", *approval.DisapproveReason), nil
	}

	// ---- 场景 1：用户批准 ----
	fmt.Println("--- 场景 1：用户批准操作 ---")
	fmt.Println()

	// 第一次运行：触发中断
	fmt.Println("[Agent] 请求删除文件: important.txt")
	_, err := deleteTool(ctx, "important.txt", nil)
	if err != nil {
		if ie, ok := err.(*InterruptError); ok {
			fmt.Printf("[系统] 需要审批: %s\n", ie.Info.Message)
			fmt.Println("[系统] 等待管理员审批...")
			fmt.Println()

			// 模拟管理员批准
			fmt.Println("[管理员] 批准操作")
			approval := &ApprovalResult{Approved: true}
			result, err := deleteTool(ctx, "important.txt", approval)
			if err != nil {
				fmt.Printf("[错误] %v\n", err)
				return
			}
			fmt.Printf("[结果] %s\n", result)
		}
	}

	fmt.Println()
	fmt.Println("--- 场景 2：用户拒绝 ---")
	fmt.Println()

	// ---- 场景 2：用户拒绝 ----
	fmt.Println("[Agent] 请求删除文件: config.json")
	_, err = deleteTool(ctx, "config.json", nil)
	if err != nil {
		if ie, ok := err.(*InterruptError); ok {
			fmt.Printf("[系统] 需要审批: %s\n", ie.Info.Message)
			fmt.Println("[系统] 等待管理员审批...")
			fmt.Println()

			// 模拟管理员拒绝
			reason := "这是配置文件，不能删除"
			fmt.Println("[管理员] 拒绝操作")
			approval := &ApprovalResult{
				Approved:         false,
				DisapproveReason: &reason,
			}
			result, err := deleteTool(ctx, "config.json", approval)
			if err != nil {
				fmt.Printf("[错误] %v\n", err)
				return
			}
			fmt.Printf("[结果] %s\n", result)
		}
	}

	fmt.Println()
	fmt.Println("=== 审批流程总结 ===")
	fmt.Println("1. 工具执行前触发 Interrupt")
	fmt.Println("2. 框架暂停执行，保存状态")
	fmt.Println("3. 管理员查看操作详情")
	fmt.Println("4. 管理员批准或拒绝")
	fmt.Println("5. 框架根据审批结果恢复或终止")
}

// --------------------------------------------------------------------------
// 示例 4：审阅编辑
// --------------------------------------------------------------------------

// 示例 4：审阅编辑
// 演示工具参数的人工审阅和修改
// 与审批不同，审阅编辑允许修改参数后再执行
func demoReviewEdit() {
	fmt.Println("=== 示例 4：审阅编辑 ===")
	fmt.Println("演示工具参数的人工审阅和修改")
	fmt.Println()

	ctx := context.Background()

	// 审阅结果
	// 在真实 Eino 中，对应 tool.ReviewEditResult
	type ReviewResult struct {
		Approved           bool    `json:"approved"`
		EditedArgumentsJSON *string `json:"edited_arguments_json,omitempty"`
		Disapproved        bool    `json:"disapproved"`
		DisapproveReason   *string `json:"disapprove_reason,omitempty"`
	}

	// 审阅编辑信息
	// 在真实 Eino 中，对应 tool.ReviewEditInfo
	type ReviewEditInfo struct {
		ToolName  string         `json:"tool_name"`
		Arguments map[string]any `json:"arguments"`
		Result    *ReviewResult  `json:"result,omitempty"`
	}

	// 创建一个带审阅编辑的工具
	sendEmailTool := func(ctx context.Context, info *ReviewEditInfo) (string, error) {
		// 如果有审阅结果
		if info.Result != nil {
			if info.Result.Disapproved {
				return fmt.Sprintf("邮件发送被拒绝: %s", *info.Result.DisapproveReason), nil
			}
			if info.Result.EditedArgumentsJSON != nil {
				return fmt.Sprintf("使用修改后的参数发送邮件: %s", *info.Result.EditedArgumentsJSON), nil
			}
			return "邮件发送成功！", nil
		}

		// 触发中断，请求审阅
		return "", &InterruptError{
			Info: &InterruptInfo{
				ID:      "review-email",
				Message: "工具 'SendEmail' 请求执行，请审阅参数",
				State: map[string]any{
					"review_info": &ReviewEditInfo{
						ToolName: "SendEmail",
						Arguments: map[string]any{
							"to":      "john@example.com",
							"subject": "会议通知",
							"body":    "明天下午3点开会，请准时参加。",
						},
					},
				},
			},
		}
	}

	// ---- 场景 1：直接批准 ----
	fmt.Println("--- 场景 1：直接批准 ---")
	fmt.Println()

	fmt.Println("[Agent] 请求发送邮件:")
	fmt.Println("  收件人: john@example.com")
	fmt.Println("  主题:   会议通知")
	fmt.Println("  内容:   明天下午3点开会，请准时参加。")
	fmt.Println()

	_, err := sendEmailTool(ctx, &ReviewEditInfo{
		ToolName: "SendEmail",
		Arguments: map[string]any{
			"to":      "john@example.com",
			"subject": "会议通知",
			"body":    "明天下午3点开会，请准时参加。",
		},
	})
	if err != nil {
		if ie, ok := err.(*InterruptError); ok {
			fmt.Printf("[系统] 需要审阅: %s\n", ie.Info.Message)
			fmt.Println()

			// 模拟人工直接批准
			fmt.Println("[人工] 参数正确，直接批准")
			result, err := sendEmailTool(ctx, &ReviewEditInfo{
				ToolName: "SendEmail",
				Arguments: map[string]any{
					"to":      "john@example.com",
					"subject": "会议通知",
					"body":    "明天下午3点开会，请准时参加。",
				},
				Result: &ReviewResult{Approved: true},
			})
			if err != nil {
				fmt.Printf("[错误] %v\n", err)
				return
			}
			fmt.Printf("[结果] %s\n", result)
		}
	}

	fmt.Println()

	// ---- 场景 2：修改参数后批准 ----
	fmt.Println("--- 场景 2：修改参数后批准 ---")
	fmt.Println()

	fmt.Println("[Agent] 请求发送邮件:")
	fmt.Println("  收件人: john@example.com")
	fmt.Println("  主题:   会议通知")
	fmt.Println("  内容:   明天下午3点开会，请准时参加。")
	fmt.Println()

	_, err = sendEmailTool(ctx, &ReviewEditInfo{
		ToolName: "SendEmail",
		Arguments: map[string]any{
			"to":      "john@example.com",
			"subject": "会议通知",
			"body":    "明天下午3点开会，请准时参加。",
		},
	})
	if err != nil {
		if _, ok := err.(*InterruptError); ok {
			fmt.Println("[系统] 需要审阅")
			fmt.Println()

			// 模拟人工修改参数
			editedJSON := `{"to":"john@example.com","subject":"紧急会议通知","body":"明天下午2点开会，请准时参加。"}`
			fmt.Println("[人工] 修改了会议时间，使用新参数")
			result, err := sendEmailTool(ctx, &ReviewEditInfo{
				ToolName: "SendEmail",
				Arguments: map[string]any{
					"to":      "john@example.com",
					"subject": "会议通知",
					"body":    "明天下午3点开会，请准时参加。",
				},
				Result: &ReviewResult{
					Approved:            true,
					EditedArgumentsJSON: &editedJSON,
				},
			})
			if err != nil {
				fmt.Printf("[错误] %v\n", err)
				return
			}
			fmt.Printf("[结果] %s\n", result)
		}
	}

	fmt.Println()

	// ---- 场景 3：拒绝 ----
	fmt.Println("--- 场景 3：拒绝操作 ---")
	fmt.Println()

	fmt.Println("[Agent] 请求发送邮件:")
	fmt.Println("  收件人: john@example.com")
	fmt.Println("  主题:   会议通知")
	fmt.Println("  内容:   明天下午3点开会，请准时参加。")
	fmt.Println()

	_, err = sendEmailTool(ctx, &ReviewEditInfo{
		ToolName: "SendEmail",
		Arguments: map[string]any{
			"to":      "john@example.com",
			"subject": "会议通知",
			"body":    "明天下午3点开会，请准时参加。",
		},
	})
	if err != nil {
		if _, ok := err.(*InterruptError); ok {
			fmt.Println("[系统] 需要审阅")
			fmt.Println()

			// 模拟人工拒绝
			reason := "会议已取消，不需要发送"
			fmt.Println("[人工] 拒绝操作")
			result, err := sendEmailTool(ctx, &ReviewEditInfo{
				ToolName: "SendEmail",
				Arguments: map[string]any{
					"to":      "john@example.com",
					"subject": "会议通知",
					"body":    "明天下午3点开会，请准时参加。",
				},
				Result: &ReviewResult{
					Disapproved:      true,
					DisapproveReason: &reason,
				},
			})
			if err != nil {
				fmt.Printf("[错误] %v\n", err)
				return
			}
			fmt.Printf("[结果] %s\n", result)
		}
	}

	fmt.Println()
	fmt.Println("=== 审阅编辑总结 ===")
	fmt.Println("1. 工具执行前触发中断，展示参数")
	fmt.Println("2. 人工可以：")
	fmt.Println("   - 直接批准（参数正确）")
	fmt.Println("   - 修改参数后批准（参数需要调整）")
	fmt.Println("   - 拒绝操作（不应该执行）")
	fmt.Println("3. 框架根据审阅结果执行或终止")
}

// --------------------------------------------------------------------------
// 示例 5：交互式 Agent
// --------------------------------------------------------------------------

// 示例 5：交互式 Agent
// 演示一个完整的交互式 Agent，支持多步中断和恢复
// 这是 Human-in-the-Loop 的完整实现
func demoInteractiveAgent() {
	fmt.Println("=== 示例 5：交互式 Agent ===")
	fmt.Println("演示一个完整的交互式 Agent，支持多步中断和恢复")
	fmt.Println("不需要 API Key，使用终端交互")
	fmt.Println()

	// 创建交互式 Agent
	agent := NewInteractiveAgent()
	scanner := bufio.NewScanner(os.Stdin)

	// 创建任务
	task := &AgentTask{
		ID:       "task-001",
		Input:    "处理用户数据并生成报告",
		Steps:    []string{"数据收集", "数据验证", "数据转换", "报告生成"},
		Current:  0,
		Results:  []string{},
		Status:   "running",
	}

	fmt.Printf("[Agent] 开始任务: %s\n", task.Input)
	fmt.Printf("[Agent] 共 %d 个步骤: %s\n", len(task.Steps), strings.Join(task.Steps, " -> "))
	fmt.Println()

	var resumeData any

	// 循环执行，直到任务完成或被终止
	for {
		// 运行 Agent
		result, signal, err := agent.Run(context.Background(), task, resumeData)
		if err != nil {
			fmt.Printf("[错误] %v\n", err)
			return
		}

		// 检查是否需要人工输入
		if signal != nil {
			fmt.Println()
			fmt.Printf("[系统] %s\n", signal.Message)
			fmt.Printf("[系统] 选项: %s\n", strings.Join(signal.Options, ", "))
			fmt.Print("[系统] 请输入: ")

			if !scanner.Scan() {
				break
			}
			userInput := strings.TrimSpace(scanner.Text())
			fmt.Println()

			// 处理用户输入
			switch strings.ToLower(userInput) {
			case "终止", "cancel", "quit":
				fmt.Println("[Agent] 任务已终止")
				return
			case "跳过", "skip":
				fmt.Println("[Agent] 跳过当前步骤")
				resumeData = "skip"
			default:
				resumeData = userInput
			}

			task = result
			continue
		}

		// 任务完成
		fmt.Println()
		fmt.Println("========================================")
		fmt.Println("          任 务 完 成")
		fmt.Println("========================================")
		fmt.Printf("任务 ID: %s\n", result.ID)
		fmt.Printf("状态: %s\n", result.Status)
		fmt.Println()
		fmt.Println("执行结果:")
		for i, r := range result.Results {
			fmt.Printf("  %d. %s\n", i+1, r)
		}
		return
	}
}

// AgentTask Agent 任务
type AgentTask struct {
	ID       string   `json:"id"`       // 任务 ID
	Input    string   `json:"input"`    // 用户输入
	Steps    []string `json:"steps"`    // 任务步骤
	Current  int      `json:"current"`  // 当前步骤
	Results  []string `json:"results"`  // 步骤结果
	Status   string   `json:"status"`   // 任务状态
}

// InterruptSignal 中断信号
type InterruptSignal struct {
	TaskID  string   `json:"task_id"`  // 任务 ID
	Message string   `json:"message"`  // 展示给用户的消息
	Options []string `json:"options"`  // 可选操作
}

// InteractiveAgent 交互式 Agent
type InteractiveAgent struct {
	store *MemoryCheckPointStore
}

// NewInteractiveAgent 创建交互式 Agent
func NewInteractiveAgent() *InteractiveAgent {
	return &InteractiveAgent{
		store: NewMemoryCheckPointStore(),
	}
}

// Run 执行任务
// 参数：
//   - ctx: 上下文
//   - task: 任务
//   - resumeData: 恢复数据（nil 表示首次运行）
//
// 返回：
//   - 更新后的任务
//   - 中断信号（如果需要人工输入）
//   - 错误
func (a *InteractiveAgent) Run(ctx context.Context, task *AgentTask, resumeData any) (*AgentTask, *InterruptSignal, error) {
	// 处理恢复数据
	if resumeData != nil {
		data := fmt.Sprintf("%v", resumeData)
		if data == "skip" {
			fmt.Printf("[Agent] 步骤 %d 已跳过\n", task.Current+1)
			task.Results = append(task.Results, fmt.Sprintf("步骤 '%s' 已跳过", task.Steps[task.Current]))
		} else {
			fmt.Printf("[Agent] 收到人工输入: %s\n", data)
			task.Results = append(task.Results, fmt.Sprintf("步骤 '%s' 完成 (人工确认: %s)", task.Steps[task.Current], data))
		}
		task.Current++
	}

	// 执行步骤
	for task.Current < len(task.Steps) {
		stepName := task.Steps[task.Current]
		fmt.Printf("[Agent] 执行步骤 %d/%d: %s\n", task.Current+1, len(task.Steps), stepName)

		// 模拟步骤执行
		time.Sleep(200 * time.Millisecond)

		// 每隔一步需要人工确认
		if task.Current == 1 {
			// 保存检查点
			task.Status = "waiting_approval"
			a.store.Save(ctx, &CheckPoint{
				ID:        task.ID,
				State:     taskToMap(task),
				Timestamp: time.Now(),
			})

			return task, &InterruptSignal{
				TaskID:  task.ID,
				Message: fmt.Sprintf("步骤 '%s' 完成，需要人工确认是否继续", stepName),
				Options: []string{"继续", "跳过", "终止"},
			}, nil
		}

		// 正常完成
		task.Results = append(task.Results, fmt.Sprintf("步骤 '%s' 完成", stepName))
		task.Current++
	}

	// 任务完成
	task.Status = "completed"
	return task, nil, nil
}

// taskToMap 将任务转换为 map（用于检查点存储）
func taskToMap(task *AgentTask) map[string]any {
	return map[string]any{
		"id":       task.ID,
		"input":    task.Input,
		"current":  task.Current,
		"status":   task.Status,
		"results":  task.Results,
	}
}

// --------------------------------------------------------------------------
// 示例 6：真实 ADK Agent
// --------------------------------------------------------------------------

// 示例 6：真实 ADK Agent
// 展示如何在真实的 Eino ADK Agent 中使用中断/恢复
// 需要设置 OPENAI_API_KEY 环境变量
func demoRealAgent() {
	fmt.Println("=== 示例 6：真实 ADK Agent 审批示例 ===")
	fmt.Println("展示如何在真实的 Eino ADK Agent 中使用中断/恢复")
	fmt.Println()

	// 检查 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("[错误] 请设置 OPENAI_API_KEY 环境变量")
		fmt.Println()
		fmt.Println("设置方式:")
		fmt.Println("  Windows:   set OPENAI_API_KEY=your-api-key-here")
		fmt.Println("  Linux/Mac: export OPENAI_API_KEY=\"your-api-key-here\"")
		fmt.Println()
		fmt.Println("提示: 其他示例不需要 API Key，可以先运行:")
		fmt.Println("  go run main.go demo")
		fmt.Println("  go run main.go checkpoint")
		fmt.Println("  go run main.go approval")
		fmt.Println("  go run main.go review")
		fmt.Println("  go run main.go interactive")
		return
	}

	fmt.Println("[提示] 真实 ADK Agent 示例需要完整的 Eino ADK 环境")
	fmt.Println("[提示] 请参考以下代码结构实现:")
	fmt.Println()

	// 展示代码结构（而不是直接运行，因为需要完整的 ADK 依赖）
	showRealAgentCode()
}

// showRealAgentCode 展示真实 ADK Agent 的代码结构
func showRealAgentCode() {
	fmt.Println(`// === 真实 ADK Agent 代码结构 ===
//
// 1. 定义工具
//
//   type sendEmailInput struct {
//       To      string ` + "`json:\"to\"`" + `
//       Subject string ` + "`json:\"subject\"`" + `
//       Body    string ` + "`json:\"body\"`" + `
//   }
//
//   func newSendEmailTool() tool.InvokableTool {
//       t, _ := utils.InferTool("SendEmail", "发送邮件",
//           func(ctx context.Context, input sendEmailInput) (string, error) {
//               return fmt.Sprintf("邮件已发送到 %s", input.To), nil
//           })
//       return t
//   }
//
// 2. 创建 Agent（使用 InvokableApprovableTool 包装工具）
//
//   sendEmail := newSendEmailTool()
//   a, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
//       Name:        "EmailAgent",
//       Description: "邮件助手",
//       Instruction: "使用 SendEmail 工具发送邮件",
//       Model:       chatModel,
//       ToolsConfig: adk.ToolsConfig{
//           ToolsNodeConfig: compose.ToolsNodeConfig{
//               Tools: []tool.BaseTool{
//                   &tool.InvokableApprovableTool{InvokableTool: sendEmail},
//               },
//           },
//       },
//   })
//
// 3. 创建 Runner（配置检查点存储）
//
//   runner := adk.NewRunner(ctx, adk.RunnerConfig{
//       Agent:           a,
//       CheckPointStore: &InMemoryStore{},
//   })
//
// 4. 运行并处理中断
//
//   iter := runner.Query(ctx, "给 john@example.com 发邮件",
//       adk.WithCheckPointID("email-001"))
//
//   var lastEvent *adk.AgentEvent
//   for {
//       event, ok := iter.Next()
//       if !ok { break }
//       lastEvent = event
//   }
//
// 5. 检查中断并恢复
//
//   if lastEvent.Action.Interrupted != nil {
//       interruptID := lastEvent.Action.Interrupted.InterruptContexts[0].ID
//
//       // 获取人工输入
//       approval := &tool.ApprovalResult{Approved: true}
//
//       // 恢复执行
//       iter, _ = runner.ResumeWithParams(ctx, "email-001",
//           &adk.ResumeParams{
//               Targets: map[string]any{interruptID: approval},
//           })
//   }
//
// === 完整示例请参考 ===
// https://github.com/cloudwego/eino-examples/tree/main/adk/human-in-the-loop`)
}

// ============================================================================
// 辅助函数
// ============================================================================

// strPtr 返回字符串的指针
// Go 中没有直接获取字符串指针的语法，需要通过函数返回
func strPtr(s string) *string {
	return &s
}

// prettyPrint 格式化输出 JSON
func prettyPrint(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
