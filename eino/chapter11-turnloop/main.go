package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ============================================================================
// 第 11 章：TurnLoop —— 抢占、中止与多轮生命周期
// ============================================================================
//
// 本示例将演示：
// 1. 基本轮次循环（Turn Loop）
// 2. 超时控制（Timeout Control）
// 3. 取消机制（Cancellation）
// 4. 抢占式调度（Preemptive Scheduling）
// 5. 优雅关闭（Graceful Shutdown）
//
// 运行方式：go run main.go
// ============================================================================

// TurnStatus 表示轮次的状态
type TurnStatus string

const (
	StatusPending   TurnStatus = "pending"   // 等待执行
	StatusRunning   TurnStatus = "running"   // 正在执行
	StatusCompleted TurnStatus = "completed" // 执行完成
	StatusCancelled TurnStatus = "cancelled" // 被取消
	StatusPreempted TurnStatus = "preempted" // 被抢占
)

// Turn 表示一轮对话
type Turn struct {
	ID        int                // 轮次 ID
	Input     string             // 用户输入
	Output    string             // AI 输出
	Status    TurnStatus         // 当前状态
	Priority  int                // 优先级（1-10，10 最高）
	CreatedAt time.Time          // 创建时间
	CompletedAt time.Time        // 完成时间
	Context   context.Context    // 该轮次的上下文（用于取消控制）
	Cancel    context.CancelFunc // 取消函数（调用它来取消这个轮次）
	mu        sync.Mutex         // 保护并发访问
}

// TurnLoop 管理多轮对话的循环
type TurnLoop struct {
	turns     []*Turn           // 所有轮次（历史记录）
	current   *Turn             // 当前正在执行的轮次
	queue     []*Turn           // 等待执行的轮次队列
	mu        sync.Mutex        // 保护共享数据
	wg        sync.WaitGroup    // 等待所有任务完成
	isRunning bool              // 是否正在运行
	logger    *log.Logger       // 日志记录器
}

// NewTurnLoop 创建一个新的轮次循环
func NewTurnLoop() *TurnLoop {
	return &TurnLoop{
		turns:     make([]*Turn, 0),
		queue:     make([]*Turn, 0),
		isRunning: true,
		logger:    log.New(os.Stdout, "[TurnLoop] ", log.LstdFlags),
	}
}

// ============================================================================
// 示例 1：基础轮次循环
// ============================================================================

// AddTurn 添加一个新的轮次到队列
func (tl *TurnLoop) AddTurn(input string, priority int) *Turn {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	// 创建带取消功能的上下文
	ctx, cancel := context.WithCancel(context.Background())

	turn := &Turn{
		ID:        len(tl.turns) + 1,
		Input:     input,
		Status:    StatusPending,
		Priority:  priority,
		CreatedAt: time.Now(),
		Context:   ctx,
		Cancel:    cancel,
	}

	tl.turns = append(tl.turns, turn)
	tl.queue = append(tl.queue, turn)

	tl.logger.Printf("添加轮次 #%d: %s (优先级: %d)", turn.ID, input, priority)
	return turn
}

// ExecuteTurn 执行单个轮次
func (tl *TurnLoop) ExecuteTurn(turn *Turn) error {
	turn.mu.Lock()
	turn.Status = StatusRunning
	turn.mu.Unlock()

	tl.logger.Printf("开始执行轮次 #%d: %s", turn.ID, turn.Input)

	// 模拟 AI 处理（会检查取消信号）
	result, err := tl.simulateAIProcessing(turn)
	if err != nil {
		turn.mu.Lock()
		turn.Status = StatusCancelled
		turn.CompletedAt = time.Now()
		turn.mu.Unlock()
		tl.logger.Printf("轮次 #%d 被取消: %v", turn.ID, err)
		return err
	}

	turn.mu.Lock()
	turn.Output = result
	turn.Status = StatusCompleted
	turn.CompletedAt = time.Now()
	turn.mu.Unlock()

	tl.logger.Printf("轮次 #%d 完成: %s", turn.ID, result)
	return nil
}

// simulateAIProcessing 模拟 AI 处理过程（会主动检查取消信号）
func (tl *TurnLoop) simulateAIProcessing(turn *Turn) (string, error) {
	// 模拟处理步骤
	steps := []string{
		"分析用户输入...",
		"检索相关知识...",
		"生成回复...",
		"优化回复...",
	}

	result := ""
	for i, step := range steps {
		// 关键：在每个步骤检查取消信号
		select {
		case <-turn.Context.Done():
			// 收到取消信号，立即返回
			return "", turn.Context.Err()
		default:
			// 继续执行
		}

		tl.logger.Printf("  轮次 #%d - 步骤 %d/%d: %s", turn.ID, i+1, len(steps), step)
		result += step + " "

		// 模拟处理时间
		time.Sleep(time.Duration(500+rand.Intn(500)) * time.Millisecond)
	}

	return fmt.Sprintf("处理完成！输入: '%s' -> 输出: '%s'", turn.Input, result), nil
}

// ============================================================================
// 示例 2：超时控制
// ============================================================================

// ExecuteWithTimeout 使用超时控制执行轮次
func (tl *TurnLoop) ExecuteWithTimeout(turn *Turn, timeout time.Duration) error {
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(turn.Context, timeout)
	defer cancel() // 确保资源被释放

	// 更新轮次的上下文
	turn.Context = ctx
	turn.Cancel = cancel

	tl.logger.Printf("轮次 #%d 开始执行（超时: %v）", turn.ID, timeout)

	// 在 goroutine 中执行任务
	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		result, err := tl.simulateAIProcessing(turn)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	// 等待任务完成或超时
	select {
	case result := <-resultCh:
		turn.mu.Lock()
		turn.Output = result
		turn.Status = StatusCompleted
		turn.CompletedAt = time.Now()
		turn.mu.Unlock()
		tl.logger.Printf("轮次 #%d 在超时前完成", turn.ID)
		return nil

	case err := <-errCh:
		turn.mu.Lock()
		turn.Status = StatusCancelled
		turn.CompletedAt = time.Now()
		turn.mu.Unlock()
		return err

	case <-ctx.Done():
		turn.mu.Lock()
		turn.Status = StatusCancelled
		turn.CompletedAt = time.Now()
		turn.mu.Unlock()
		tl.logger.Printf("轮次 #%d 超时: %v", turn.ID, ctx.Err())
		return ctx.Err()
	}
}

// ============================================================================
// 示例 3：取消机制
// ============================================================================

// CancelTurn 主动取消正在执行的轮次
func (tl *TurnLoop) CancelTurn(turnID int) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	// 查找轮次
	for _, turn := range tl.turns {
		if turn.ID == turnID {
			turn.mu.Lock()
			defer turn.mu.Unlock()

			// 检查是否可以取消
			if turn.Status != StatusRunning && turn.Status != StatusPending {
				return fmt.Errorf("轮次 #%d 状态为 %s，无法取消", turnID, turn.Status)
			}

			// 调用取消函数
			if turn.Cancel != nil {
				turn.Cancel()
				tl.logger.Printf("已发送取消信号给轮次 #%d", turnID)
				return nil
			}
			return fmt.Errorf("轮次 #%d 没有取消函数", turnID)
		}
	}
	return fmt.Errorf("找不到轮次 #%d", turnID)
}

// CancelAllCancelled 取消所有等待中的轮次
func (tl *TurnLoop) CancelAllPending() {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	count := 0
	for _, turn := range tl.turns {
		if turn.Status == StatusPending {
			if turn.Cancel != nil {
				turn.Cancel()
				turn.Status = StatusCancelled
				count++
			}
		}
	}
	tl.logger.Printf("已取消 %d 个等待中的轮次", count)
}

// ============================================================================
// 示例 4：抢占式调度
// ============================================================================

// PreemptCurrent 抢占当前正在执行的轮次
func (tl *TurnLoop) PreemptCurrent(newTurn *Turn) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	// 检查是否有正在运行的任务
	if tl.current != nil && tl.current.Status == StatusRunning {
		// 检查新任务的优先级是否更高
		if newTurn.Priority <= tl.current.Priority {
			return fmt.Errorf("新任务优先级 (%d) 不高于当前任务 (%d)，无法抢占",
				newTurn.Priority, tl.current.Priority)
		}

		// 抢占当前任务
		tl.logger.Printf("轮次 #%d (优先级 %d) 抢占轮次 #%d (优先级 %d)",
			newTurn.ID, newTurn.Priority, tl.current.ID, tl.current.Priority)

		// 取消当前任务
		tl.current.Cancel()
		tl.current.Status = StatusPreempted

		// 将被抢占的任务放回队列（可以稍后恢复）
		tl.queue = append([]*Turn{tl.current}, tl.queue...)
	}

	// 设置新任务为当前任务
	newTurn.Status = StatusRunning
	tl.current = newTurn

	return nil
}

// ScheduleWithPriority 带优先级的调度
func (tl *TurnLoop) ScheduleWithPriority(turn *Turn) error {
	// 尝试抢占
	err := tl.PreemptCurrent(turn)
	if err != nil {
		// 无法抢占，添加到队列
		tl.logger.Printf("轮次 #%d 无法抢占，添加到队列", turn.ID)
		return nil
	}

	// 执行新任务
	go func() {
		tl.ExecuteTurn(turn)

		// 任务完成后，检查是否有被抢占的任务需要恢复
		tl.restorePreemptedTurn()
	}()

	return nil
}

// restorePreemptedTurn 恢复被抢占的任务
func (tl *TurnLoop) restorePreemptedTurn() {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	// 查找被抢占的任务
	for i, turn := range tl.queue {
		if turn.Status == StatusPreempted {
			tl.logger.Printf("恢复被抢占的轮次 #%d", turn.ID)

			// 从队列中移除
			tl.queue = append(tl.queue[:i], tl.queue[i+1:]...)

			// 创建新的上下文（旧的已经被取消）
			ctx, cancel := context.WithCancel(context.Background())
			turn.Context = ctx
			turn.Cancel = cancel
			turn.Status = StatusPending

			// 设置为当前任务
			tl.current = turn

			// 执行任务
			go tl.ExecuteTurn(turn)
			return
		}
	}
}

// ============================================================================
// 示例 5：优雅关闭
// ============================================================================

// GracefulShutdown 优雅关闭 TurnLoop
func (tl *TurnLoop) GracefulShutdown(timeout time.Duration) error {
	tl.logger.Printf("开始优雅关闭（超时: %v）", timeout)

	// 1. 停止接受新任务
	tl.mu.Lock()
	tl.isRunning = false
	tl.mu.Unlock()

	// 2. 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 3. 等待所有任务完成或超时
	done := make(chan struct{})
	go func() {
		tl.wg.Wait() // 等待所有任务完成
		close(done)
	}()

	select {
	case <-done:
		tl.logger.Printf("所有任务已完成")
	case <-ctx.Done():
		tl.logger.Printf("等待超时，强制取消剩余任务")
		tl.CancelAllPending()
	}

	// 4. 打印最终状态
	tl.PrintStatus()

	return nil
}

// ============================================================================
// 辅助函数
// ============================================================================

// PrintStatus 打印所有轮次的状态
func (tl *TurnLoop) PrintStatus() {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	fmt.Println("\n" + "="*60)
	fmt.Println("轮次状态汇总")
	fmt.Println("="*60)

	for _, turn := range tl.turns {
		duration := ""
		if !turn.CompletedAt.IsZero() {
			duration = turn.CompletedAt.Sub(turn.CreatedAt).String()
		}

		fmt.Printf("轮次 #%d | 状态: %-10s | 优先级: %d | 耗时: %s\n",
			turn.ID, turn.Status, turn.Priority, duration)
		fmt.Printf("  输入: %s\n", turn.Input)
		if turn.Output != "" {
			fmt.Printf("  输出: %s\n", turn.Output)
		}
		fmt.Println("-" * 60)
	}

	// 统计
	completed := 0
	cancelled := 0
	preempted := 0
	for _, turn := range tl.turns {
		switch turn.Status {
		case StatusCompleted:
			completed++
		case StatusCancelled:
			cancelled++
		case StatusPreempted:
			preempted++
		}
	}

	fmt.Printf("\n统计: 完成 %d | 取消 %d | 被抢占 %d | 总计 %d\n",
		completed, cancelled, preempted, len(tl.turns))
	fmt.Println("="*60)
}

// ============================================================================
// 主函数：演示所有功能
// ============================================================================

func main() {
	fmt.Println("============================================================")
	fmt.Println("第 11 章：TurnLoop —— 抢占、中止与多轮生命周期")
	fmt.Println("============================================================")
	fmt.Println()

	// 创建 TurnLoop
	loop := NewTurnLoop()

	// ========================================================================
	// 演示 1：基础轮次循环
	// ========================================================================
	fmt.Println("\n>>> 演示 1：基础轮次循环")
	fmt.Println("-"*60)

	// 添加几个轮次
	loop.AddTurn("你好，今天天气怎么样？", 5)
	loop.AddTurn("帮我写一首诗", 3)
	loop.AddTurn("什么是机器学习？", 7)

	// 执行所有轮次
	fmt.Println("\n执行所有轮次...")
	for _, turn := range loop.turns {
		loop.wg.Add(1)
		go func(t *Turn) {
			defer loop.wg.Done()
			loop.ExecuteTurn(t)
		}(turn)
	}

	loop.wg.Wait()
	loop.PrintStatus()

	// ========================================================================
	// 演示 2：超时控制
	// ========================================================================
	fmt.Println("\n>>> 演示 2：超时控制")
	fmt.Println("-"*60)

	// 创建一个新的 TurnLoop
	loop2 := NewTurnLoop()

	// 添加一个任务，但设置很短的超时时间
	turn := loop2.AddTurn("这个任务会超时", 5)

	// 设置超时时间为 1 秒（任务需要 2-4 秒完成）
	fmt.Println("设置超时时间为 1 秒...")
	loop2.ExecuteWithTimeout(turn, 1*time.Second)

	loop2.PrintStatus()

	// ========================================================================
	// 演示 3：主动取消
	// ========================================================================
	fmt.Println("\n>>> 演示 3：主动取消")
	fmt.Println("-"*60)

	loop3 := NewTurnLoop()

	// 添加一个长时间运行的任务
	longTask := loop3.AddTurn("这是一个长时间运行的任务", 5)

	// 在后台执行任务
	go func() {
		loop3.ExecuteTurn(longTask)
	}()

	// 等待一会儿，然后取消
	time.Sleep(1 * time.Second)
	fmt.Println("\n主动取消任务...")
	loop3.CancelTurn(longTask.ID)

	// 等待任务处理取消信号
	time.Sleep(500 * time.Millisecond)
	loop3.PrintStatus()

	// ========================================================================
	// 演示 4：抢占式调度
	// ========================================================================
	fmt.Println("\n>>> 演示 4：抢占式调度")
	fmt.Println("-"*60)

	loop4 := NewTurnLoop()

	// 添加一个低优先级任务
	lowPriority := loop4.AddTurn("低优先级任务", 3)

	// 开始执行低优先级任务
	go func() {
		loop4.ExecuteTurn(lowPriority)
	}()

	// 等待一会儿
	time.Sleep(1 * time.Second)

	// 添加一个高优先级任务，它会抢占低优先级任务
	highPriority := &Turn{
		ID:       99,
		Input:    "紧急任务！需要立即处理",
		Priority: 9,
	}
	highPriority.Context, highPriority.Cancel = context.WithCancel(context.Background())

	fmt.Println("\n添加高优先级任务，抢占低优先级任务...")
	loop4.ScheduleWithPriority(highPriority)

	// 等待所有任务完成
	time.Sleep(5 * time.Second)
	loop4.PrintStatus()

	// ========================================================================
	// 演示 5：优雅关闭
	// ========================================================================
	fmt.Println("\n>>> 演示 5：优雅关闭")
	fmt.Println("-"*60)

	loop5 := NewTurnLoop()

	// 添加几个任务
	loop5.AddTurn("任务 A", 5)
	loop5.AddTurn("任务 B", 3)
	loop5.AddTurn("任务 C", 7)

	// 开始执行任务
	for _, t := range loop5.turns {
		loop5.wg.Add(1)
		go func(turn *Turn) {
			defer loop5.wg.Done()
			loop5.ExecuteTurn(turn)
		}(t)
	}

	// 监听系统信号（Ctrl+C）
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 模拟：2 秒后发送关闭信号
	go func() {
		time.Sleep(2 * time.Second)
		sigCh <- syscall.SIGTERM
	}()

	// 等待关闭信号
	sig := <-sigCh
	fmt.Printf("\n收到信号: %v\n", sig)

	// 执行优雅关闭（等待 3 秒）
	loop5.GracefulShutdown(3 * time.Second)

	fmt.Println("\n程序结束")
}
