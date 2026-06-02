// ============================================================================
// 第 12 章：RAG（Retrieval-Augmented Generation）—— 检索增强生成
// ============================================================================
//
// 本文件演示 Eino 框架的 RAG 核心概念和完整流程。
//
// 由于 RAG 需要向量数据库和 Embedding 模型等外部依赖，
// 本示例采用"先理解原理，再看实际用法"的方式：
//
//   Part 1: 用纯 Go 模拟 RAG 的核心概念（不需要 API Key）
//   Part 2: 展示如何在真实 Eino 中构建 RAG 应用
//
// 运行方式：
//   go run main.go              - 查看所有可用命令
//   go run main.go demo         - 运行完整 RAG 流程演示（模拟，不需要 API Key）
//   go run main.go document     - 文档加载和分割示例
//   go run main.go embedding    - Embedding 向量嵌入示例
//   go run main.go retriever    - Retriever 检索示例
//   go run main.go template     - ChatTemplate 构建提示词示例
//   go run main.go rag          - 完整 RAG 应用示例（模拟 Embedding）
//   go run main.go interactive  - 交互式 RAG 查询
//
// ============================================================================

package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"sort"
	"strings"
)

// ============================================================================
// Part 1: 模拟 RAG 核心组件（帮助理解原理）
// ============================================================================
//
// 这部分用纯 Go 代码模拟 Eino 的 RAG 组件。
// 不需要任何外部依赖，帮助你理解 RAG 的核心思想。
//
// 在真实 Eino 中，这些组件由 eino-ext 扩展包提供：
//   - github.com/cloudwego/eino-ext/components/embedding/openai
//   - github.com/cloudwego/eino-ext/components/indexer/milvus
//   - github.com/cloudwego/eino-ext/components/retriever/milvus
//
// ============================================================================

// --------------------------------------------------------------------------
// 1.1 Document 文档类型
// --------------------------------------------------------------------------
// 在真实 Eino 中，使用 github.com/cloudwego/eino/schema.Document
// 这里我们定义一个简化版本来演示核心概念

// Document 文档结构体
// 在 Eino 中对应 schema.Document
type Document struct {
	ID       string         // 文档唯一标识
	Content  string         // 文档内容
	MetaData map[string]any // 元数据（来源、分数等）
}

// NewDocument 创建新文档
func NewDocument(id, content string) *Document {
	return &Document{
		ID:       id,
		Content:  content,
		MetaData: make(map[string]any),
	}
}

// WithScore 设置相关性分数
func (d *Document) WithScore(score float64) *Document {
	d.MetaData["_score"] = score
	return d
}

// Score 获取相关性分数
func (d *Document) Score() float64 {
	score, ok := d.MetaData["_score"].(float64)
	if !ok {
		return 0
	}
	return score
}

// WithSource 设置文档来源
func (d *Document) WithSource(source string) *Document {
	d.MetaData["source"] = source
	return d
}

// --------------------------------------------------------------------------
// 1.2 Loader 文档加载器
// --------------------------------------------------------------------------
// 在真实 Eino 中，使用 document.Loader 接口
// 这里实现一个简单的文本加载器

// Loader 文档加载器接口
// 在 Eino 中对应 document.Loader
type Loader interface {
	Load(ctx context.Context, uri string) ([]*Document, error)
}

// TextLoader 文本加载器实现
type TextLoader struct{}

// Load 从文本内容加载文档
func (l *TextLoader) Load(ctx context.Context, content string) ([]*Document, error) {
	doc := NewDocument(
		generateID(content),
		content,
	)
	doc.WithSource("text_input")
	return []*Document{doc}, nil
}

// --------------------------------------------------------------------------
// 1.3 Transformer 文档转换器（文本分割）
// --------------------------------------------------------------------------
// 在真实 Eino 中，使用 document.Transformer 接口
// 这里实现一个简单的文本分割器

// Transformer 文档转换器接口
// 在 Eino 中对应 document.Transformer
type Transformer interface {
	Transform(ctx context.Context, docs []*Document) ([]*Document, error)
}

// SimpleTextSplitter 简单的文本分割器
// 按照固定字符数分割文档，支持重叠
type SimpleTextSplitter struct {
	ChunkSize    int // 每个块的最大字符数
	ChunkOverlap int // 块之间的重叠字符数
}

// Transform 将文档分割成多个小块
func (s *SimpleTextSplitter) Transform(ctx context.Context, docs []*Document) ([]*Document, error) {
	var allChunks []*Document

	for _, doc := range docs {
		chunks := s.splitDocument(doc)
		allChunks = append(allChunks, chunks...)
	}

	return allChunks, nil
}

// splitDocument 将单个文档分割成多个块
func (s *SimpleTextSplitter) splitDocument(doc *Document) []*Document {
	var chunks []*Document
	runes := []rune(doc.Content) // 使用 rune 切片正确处理中文等多字节字符

	// 按照 ChunkSize 分割，每次前进 ChunkSize - ChunkOverlap
	for i := 0; i < len(runes); i += s.ChunkSize - s.ChunkOverlap {
		end := i + s.ChunkSize
		if end > len(runes) {
			end = len(runes)
		}

		chunk := NewDocument(
			fmt.Sprintf("%s_chunk_%d", doc.ID, len(chunks)),
			string(runes[i:end]),
		)
		chunk.WithSource(doc.ID)
		chunk.MetaData["chunk_index"] = len(chunks)

		chunks = append(chunks, chunk)

		// 如果已经到达文档末尾，停止分割
		if end == len(runes) {
			break
		}
	}

	return chunks
}

// --------------------------------------------------------------------------
// 1.4 Embedder 向量嵌入器
// --------------------------------------------------------------------------
// 在真实 Eino 中，使用 embedding.Embedder 接口
// 这里实现一个简化的向量嵌入器（基于词频哈希）
//
// 注意：这是模拟实现，仅用于学习！
// 真实项目中应使用 OpenAI text-embedding-3-small、Cohere embed 等模型

// Embedder 向量嵌入器接口
// 在 Eino 中对应 embedding.Embedder
type Embedder interface {
	EmbedStrings(ctx context.Context, texts []string) ([][]float64, error)
}

// SimpleEmbedder 简单的向量嵌入器
// 使用词频哈希将文本转换为向量
// 这种方法不理解语义，仅用于演示向量检索的原理
type SimpleEmbedder struct {
	Dimension int // 向量维度
}

// NewSimpleEmbedder 创建简单嵌入器
func NewSimpleEmbedder(dimension int) *SimpleEmbedder {
	return &SimpleEmbedder{Dimension: dimension}
}

// EmbedStrings 将多个文本转换为向量
func (e *SimpleEmbedder) EmbedStrings(ctx context.Context, texts []string) ([][]float64, error) {
	vectors := make([][]float64, len(texts))
	for i, text := range texts {
		vectors[i] = e.textToVector(text)
	}
	return vectors, nil
}

// textToVector 将单个文本转换为向量
// 使用简化的词频哈希方法
func (e *SimpleEmbedder) textToVector(text string) []float64 {
	vector := make([]float64, e.Dimension)

	// 分词（简单按空格分割）
	words := tokenize(text)

	// 统计词频并映射到向量维度
	for _, word := range words {
		// 使用 FNV 哈希将词映射到向量维度
		hash := fnv.New32a()
		hash.Write([]byte(word))
		idx := int(hash.Sum32()) % e.Dimension
		if idx < 0 {
			idx = -idx
		}
		vector[idx] += 1.0
	}

	// L2 归一化（使向量长度为 1）
	norm := 0.0
	for _, v := range vector {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vector {
			vector[i] /= norm
		}
	}

	return vector
}

// tokenize 简单的分词函数
// 将文本转小写，按空格和标点分割
func tokenize(text string) []string {
	// 转小写
	text = strings.ToLower(text)

	// 替换标点为空格
	replacer := strings.NewReplacer(
		"，", " ", "。", " ", "！", " ", "？", " ",
		"、", " ", "；", " ", "：", " ",
		",", " ", ".", " ", "!", " ", "?", " ",
		";", " ", ":", " ", "(", " ", ")", " ",
		"[", " ", "]", " ", "{", " ", "}", " ",
		"\n", " ", "\t", " ", "  ", " ",
	)
	text = replacer.Replace(text)

	// 按空格分割
	return strings.Fields(text)
}

// --------------------------------------------------------------------------
// 1.5 Indexer 索引器
// --------------------------------------------------------------------------
// 在真实 Eino 中，使用 indexer.Indexer 接口
// 这里实现一个内存索引器

// Indexer 索引器接口
// 在 Eino 中对应 indexer.Indexer
type Indexer interface {
	Store(ctx context.Context, docs []*Document, vectors [][]float64) ([]string, error)
}

// MemoryIndexer 内存索引器
// 将文档和向量存储在内存中
type MemoryIndexer struct {
	documents []*Document
	vectors   [][]float64
}

// NewMemoryIndexer 创建内存索引器
func NewMemoryIndexer() *MemoryIndexer {
	return &MemoryIndexer{
		documents: make([]*Document, 0),
		vectors:   make([][]float64, 0),
	}
}

// Store 存储文档和向量
func (idx *MemoryIndexer) Store(ctx context.Context, docs []*Document, vectors [][]float64) ([]string, error) {
	if len(docs) != len(vectors) {
		return nil, fmt.Errorf("文档数量 (%d) 与向量数量 (%d) 不匹配", len(docs), len(vectors))
	}

	ids := make([]string, len(docs))
	for i, doc := range docs {
		idx.documents = append(idx.documents, doc)
		idx.vectors = append(idx.vectors, vectors[i])
		ids[i] = doc.ID
	}

	return ids, nil
}

// GetDocuments 获取所有文档
func (idx *MemoryIndexer) GetDocuments() []*Document {
	return idx.documents
}

// GetVectors 获取所有向量
func (idx *MemoryIndexer) GetVectors() [][]float64 {
	return idx.vectors
}

// --------------------------------------------------------------------------
// 1.6 Retriever 检索器
// --------------------------------------------------------------------------
// 在真实 Eino 中，使用 retriever.Retriever 接口
// 这里实现一个基于余弦相似度的检索器

// Retriever 检索器接口
// 在 Eino 中对应 retriever.Retriever
type Retriever interface {
	Retrieve(ctx context.Context, query string, topK int) ([]*Document, error)
}

// MemoryRetriever 内存检索器
// 基于余弦相似度在内存中检索文档
type MemoryRetriever struct {
	indexer  *MemoryIndexer
	embedder *SimpleEmbedder
}

// NewMemoryRetriever 创建内存检索器
func NewMemoryRetriever(indexer *MemoryIndexer, embedder *SimpleEmbedder) *MemoryRetriever {
	return &MemoryRetriever{
		indexer:  indexer,
		embedder: embedder,
	}
}

// Retrieve 检索与查询最相关的文档
func (r *MemoryRetriever) Retrieve(ctx context.Context, query string, topK int) ([]*Document, error) {
	// 1. 将查询文本转换为向量
	queryVectors, err := r.embedder.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("查询向量化失败: %w", err)
	}
	queryVector := queryVectors[0]

	// 2. 获取所有文档和向量
	documents := r.indexer.GetDocuments()
	vectors := r.indexer.GetVectors()

	if len(documents) == 0 {
		return nil, fmt.Errorf("索引中没有文档，请先调用 IndexDocuments")
	}

	// 3. 计算每个文档与查询的相似度
	type docScore struct {
		doc   *Document
		score float64
	}

	scores := make([]docScore, len(documents))
	for i, doc := range documents {
		similarity := cosineSimilarity(queryVector, vectors[i])
		scores[i] = docScore{
			doc:   doc,
			score: similarity,
		}
	}

	// 4. 按相似度降序排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// 5. 返回前 topK 个结果
	results := make([]*Document, 0, topK)
	for i := 0; i < topK && i < len(scores); i++ {
		// 设置文档的相关性分数
		scores[i].doc.WithScore(scores[i].score)
		results = append(results, scores[i].doc)
	}

	return results, nil
}

// --------------------------------------------------------------------------
// 1.7 ChatTemplate 聊天模板
// --------------------------------------------------------------------------
// 在真实 Eino 中，使用 prompt.DefaultChatTemplate
// 这里实现一个简单的模板引擎

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string // system, user, assistant
	Content string
}

// ChatTemplate 聊天模板
type ChatTemplate struct {
	SystemPrompt string
	UserPrompt   string
}

// Format 格式化模板，替换变量
func (t *ChatTemplate) Format(variables map[string]any) []*ChatMessage {
	messages := []*ChatMessage{}

	// 添加系统消息
	if t.SystemPrompt != "" {
		systemContent := t.SystemPrompt
		for key, value := range variables {
			placeholder := fmt.Sprintf("{%s}", key)
			systemContent = strings.ReplaceAll(systemContent, placeholder, fmt.Sprintf("%v", value))
		}
		messages = append(messages, &ChatMessage{
			Role:    "system",
			Content: systemContent,
		})
	}

	// 添加用户消息
	if t.UserPrompt != "" {
		userContent := t.UserPrompt
		for key, value := range variables {
			placeholder := fmt.Sprintf("{%s}", key)
			userContent = strings.ReplaceAll(userContent, placeholder, fmt.Sprintf("%v", value))
		}
		messages = append(messages, &ChatMessage{
			Role:    "user",
			Content: userContent,
		})
	}

	return messages
}

// --------------------------------------------------------------------------
// 1.8 RAG Application 完整应用
// --------------------------------------------------------------------------
// 将所有组件组合成完整的 RAG 应用

// RAGApplication RAG 应用
type RAGApplication struct {
	loader    *TextLoader
	splitter  *SimpleTextSplitter
	embedder  *SimpleEmbedder
	indexer   *MemoryIndexer
	retriever *MemoryRetriever
	template  *ChatTemplate
}

// NewRAGApplication 创建 RAG 应用
func NewRAGApplication() *RAGApplication {
	// 创建组件
	embedder := NewSimpleEmbedder(128) // 使用 128 维向量
	indexer := NewMemoryIndexer()
	retriever := NewMemoryRetriever(indexer, embedder)

	// 创建 RAG 提示词模板
	template := &ChatTemplate{
		SystemPrompt: `你是一个智能助手。请基于以下参考资料回答用户的问题。

要求：
1. 如果参考资料中有相关信息，请基于资料回答
2. 如果参考资料中没有相关信息，请如实说明
3. 回答要简洁明了

参考资料：
{context}`,
		UserPrompt: "{question}",
	}

	return &RAGApplication{
		loader:    &TextLoader{},
		splitter:  &SimpleTextSplitter{ChunkSize: 200, ChunkOverlap: 50},
		embedder:  embedder,
		indexer:   indexer,
		retriever: retriever,
		template:  template,
	}
}

// IndexDocuments 索引文档（离线阶段）
// 这是 RAG 的第一步：将文档加载、分割、向量化并存储
func (app *RAGApplication) IndexDocuments(ctx context.Context, documents []string) error {
	fmt.Println("=== 开始索引文档 ===")

	// 步骤 1：加载文档
	fmt.Println("\n[步骤 1/4] 加载文档...")
	var allDocs []*Document
	for i, content := range documents {
		docs, err := app.loader.Load(ctx, content)
		if err != nil {
			return fmt.Errorf("加载文档 %d 失败: %w", i, err)
		}
		allDocs = append(allDocs, docs...)
	}
	fmt.Printf("  加载了 %d 个文档\n", len(allDocs))

	// 步骤 2：分割文档
	fmt.Println("\n[步骤 2/4] 分割文档...")
	chunks, err := app.splitter.Transform(ctx, allDocs)
	if err != nil {
		return fmt.Errorf("分割文档失败: %w", err)
	}
	fmt.Printf("  分割成 %d 个文本块\n", len(chunks))

	// 显示分割结果
	for i, chunk := range chunks {
		fmt.Printf("  块 %d: %s\n", i+1, truncateString(chunk.Content, 30))
	}

	// 步骤 3：向量化
	fmt.Println("\n[步骤 3/4] 生成向量嵌入...")
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}
	vectors, err := app.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		return fmt.Errorf("向量化失败: %w", err)
	}
	fmt.Printf("  生成了 %d 个向量（维度: %d）\n", len(vectors), len(vectors[0]))

	// 步骤 4：存储到索引
	fmt.Println("\n[步骤 4/4] 存储到向量索引...")
	ids, err := app.indexer.Store(ctx, chunks, vectors)
	if err != nil {
		return fmt.Errorf("存储失败: %w", err)
	}
	fmt.Printf("  成功存储 %d 个文档\n", len(ids))

	fmt.Println("\n=== 文档索引完成 ===")
	return nil
}

// Query 查询（在线阶段）
// 这是 RAG 的第二步：检索相关文档并生成回答
func (app *RAGApplication) Query(ctx context.Context, question string, topK int) (string, []*Document, error) {
	fmt.Println("\n=== 开始查询 ===")
	fmt.Printf("问题: %s\n", question)

	// 步骤 1：检索相关文档
	fmt.Println("\n[步骤 1/2] 检索相关文档...")
	docs, err := app.retriever.Retrieve(ctx, question, topK)
	if err != nil {
		return "", nil, fmt.Errorf("检索失败: %w", err)
	}
	fmt.Printf("  检索到 %d 个相关文档\n", len(docs))

	// 显示检索结果
	for i, doc := range docs {
		fmt.Printf("  文档 %d (相关度: %.4f): %s\n", i+1, doc.Score(), truncateString(doc.Content, 40))
	}

	// 步骤 2：构建带上下文的提示词
	fmt.Println("\n[步骤 2/2] 构建提示词...")

	// 将检索到的文档组合成上下文
	var contextParts []string
	for i, doc := range docs {
		contextParts = append(contextParts,
			fmt.Sprintf("[参考资料 %d] (相关度: %.2f)\n%s",
				i+1, doc.Score(), doc.Content))
	}
	contextText := strings.Join(contextParts, "\n\n")

	// 使用模板格式化
	messages := app.template.Format(map[string]any{
		"context":  contextText,
		"question": question,
	})

	// 输出构建的提示词
	fmt.Println("\n构建的提示词:")
	fmt.Println("---")
	for _, msg := range messages {
		fmt.Printf("[%s]\n%s\n\n", msg.Role, msg.Content)
	}
	fmt.Println("---")

	fmt.Println("\n=== 查询完成 ===")

	// 在没有真实 LLM 的情况下，返回检索结果摘要
	answer := buildFallbackAnswer(question, docs)
	return answer, docs, nil
}

// ============================================================================
// 辅助函数
// ============================================================================

// cosineSimilarity 计算两个向量的余弦相似度
// 余弦相似度 = (A · B) / (|A| × |B|)
// 值范围 [-1, 1]，1 表示完全相同，0 表示无关，-1 表示完全相反
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	dotProduct := 0.0 // 点积
	normA := 0.0      // 向量 A 的模
	normB := 0.0      // 向量 B 的模

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (normA * normB)
}

// truncateString 安全地截断字符串（支持中文等多字节字符）
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// generateID 生成简单的文档 ID
func generateID(content string) string {
	hash := fnv.New32a()
	hash.Write([]byte(content))
	return fmt.Sprintf("doc_%08x", hash.Sum32())
}

// buildFallbackAnswer 在没有 LLM 的情况下，构建一个基于检索结果的回答
func buildFallbackAnswer(question string, docs []*Document) string {
	if len(docs) == 0 {
		return "抱歉，没有找到与问题相关的参考资料。"
	}

	var answer strings.Builder
	answer.WriteString(fmt.Sprintf("根据检索到的 %d 个相关文档，以下是找到的信息：\n\n", len(docs)))

	for i, doc := range docs {
		answer.WriteString(fmt.Sprintf("【参考资料 %d】(相关度: %.2f)\n", i+1, doc.Score()))
		answer.WriteString(doc.Content)
		answer.WriteString("\n\n")
	}

	answer.WriteString("注：以上是基于检索的参考资料摘要。如需更智能的回答，请配置 LLM（如 OpenAI）。")
	return answer.String()
}

// ============================================================================
// 示例函数
// ============================================================================

// demoRAGFlow 演示完整的 RAG 流程
func demoRAGFlow() {
	ctx := context.Background()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║          RAG（检索增强生成）完整流程演示                      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	// 创建 RAG 应用
	app := NewRAGApplication()

	// 准备知识库文档
	knowledgeBase := []string{
		`Go 语言（Golang）是 Google 开发的一种静态强类型、编译型、并发型语言。
Go 语言的特点包括：简洁的语法、强大的并发支持（goroutine 和 channel）、
快速的编译速度、垃圾回收机制、丰富的标准库。
Go 语言非常适合构建网络服务、微服务、云原生应用和分布式系统。`,

		`Python 是一种高级、解释型、通用编程语言。
Python 的特点包括：简洁易读的语法、丰富的第三方库、动态类型系统。
Python 广泛应用于数据科学、机器学习、Web 开发、自动化脚本等领域。
常用的 Python 框架包括 Django、Flask、FastAPI 等。`,

		`Rust 是一种系统编程语言，注重安全、并发和性能。
Rust 的核心特性包括：所有权系统、零成本抽象、内存安全保证、无数据竞争。
Rust 适合构建系统软件、嵌入式开发、WebAssembly 和高性能应用。
Rust 的包管理器 Cargo 是其重要的开发工具。`,

		`Eino 是 CloudWeGo 团队开发的 Go 语言 LLM 应用开发框架。
Eino 提供了 ChatModel、Tool、Retriever、Embedding 等组件抽象。
Eino 的核心概念包括：Component（组件）、Chain（链）、Graph（图）。
Eino 支持构建 RAG 应用、Agent 应用和复杂的 AI 工作流。`,

		`RAG（Retrieval-Augmented Generation，检索增强生成）是一种结合检索和生成的 AI 技术。
RAG 的核心思想是：先从知识库中检索相关文档，再将文档作为上下文提供给 LLM 生成回答。
RAG 的优势包括：减少幻觉、知识可更新、可追溯来源、成本低于微调。
RAG 的典型流程：文档加载 → 分割 → 向量化 → 存储 → 检索 → 生成。`,
	}

	// 索引文档
	err := app.IndexDocuments(ctx, knowledgeBase)
	if err != nil {
		fmt.Printf("索引文档失败: %v\n", err)
		return
	}

	// 测试查询
	questions := []string{
		"Go 语言有什么特点？",
		"Eino 框架是什么？",
		"RAG 技术有什么优势？",
	}

	for _, question := range questions {
		fmt.Println("\n" + strings.Repeat("=", 60))
		answer, _, err := app.Query(ctx, question, 3)
		if err != nil {
			fmt.Printf("查询失败: %v\n", err)
			continue
		}

		fmt.Println("\n📋 最终回答:")
		fmt.Println(answer)
	}
}

// demoDocumentLoading 演示文档加载和分割
func demoDocumentLoading() {
	ctx := context.Background()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              文档加载和分割示例                              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	// 示例文档
	longText := `Go 语言（Golang）是 Google 于 2009 年发布的开源编程语言。
Go 语言的设计目标是结合 Python 的开发效率和 C++ 的运行效率。
Go 语言的核心特性包括简洁的语法、强大的并发支持和快速的编译速度。

Go 语言的并发模型基于 CSP（Communicating Sequential Processes）理论。
goroutine 是 Go 语言中的轻量级线程，可以轻松创建数万个并发任务。
channel 是 goroutine 之间通信的管道，实现了安全的数据传递。

Go 语言的标准库非常丰富，包括网络编程、文件操作、加密、压缩等功能。
常用的 Go 框架包括 Gin（Web 框架）、gRPC（RPC 框架）和 Eino（AI 框架）。

Go 语言的编译速度非常快，通常只需要几秒钟就能编译完成。
Go 语言生成的是静态链接的二进制文件，部署非常方便。
Go 语言的垃圾回收机制经过多次优化，已经能够满足大多数应用场景。`

	// 1. 加载文档
	fmt.Println("\n[1] 加载文档:")
	loader := &TextLoader{}
	docs, _ := loader.Load(ctx, longText)
	fmt.Printf("  加载了 %d 个文档\n", len(docs))
	fmt.Printf("  文档长度: %d 个字符\n", len(docs[0].Content))

	// 2. 分割文档
	fmt.Println("\n[2] 分割文档（ChunkSize=150, Overlap=30）:")
	splitter := &SimpleTextSplitter{
		ChunkSize:    150,
		ChunkOverlap: 30,
	}
	chunks, _ := splitter.Transform(ctx, docs)
	fmt.Printf("  分割成 %d 个文本块\n", len(chunks))

	for i, chunk := range chunks {
		fmt.Printf("\n  --- 块 %d (长度: %d) ---\n", i+1, len(chunk.Content))
		fmt.Printf("  %s\n", chunk.Content)
	}
}

// demoEmbedding 演示向量嵌入
func demoEmbedding() {
	ctx := context.Background()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              Embedding 向量嵌入示例                         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	embedder := NewSimpleEmbedder(8) // 使用 8 维向量便于展示

	texts := []string{
		"Go 语言是一种高效的编程语言",
		"Golang 是 Google 开发的语言",
		"Python 适合数据科学",
	}

	fmt.Println("\n输入文本:")
	for i, text := range texts {
		fmt.Printf("  %d. %s\n", i+1, text)
	}

	// 生成向量
	vectors, _ := embedder.EmbedStrings(ctx, texts)

	fmt.Println("\n生成的向量（8 维）:")
	for i, vec := range vectors {
		fmt.Printf("  文本 %d: [", i+1)
		for j, v := range vec {
			if j > 0 {
				fmt.Printf(", ")
			}
			fmt.Printf("%.4f", v)
		}
		fmt.Println("]")
	}

	// 计算相似度
	fmt.Println("\n相似度矩阵:")
	fmt.Printf("  %-12s", "")
	for i := range texts {
		fmt.Printf("文本 %-6d", i+1)
	}
	fmt.Println()

	for i := range texts {
		fmt.Printf("  文本 %-6d", i+1)
		for j := range texts {
			sim := cosineSimilarity(vectors[i], vectors[j])
			fmt.Printf("%-10.4f", sim)
		}
		fmt.Println()
	}

	fmt.Println("\n说明:")
	fmt.Println("  - 文本 1 和文本 2 都关于 Go 语言，相似度较高")
	fmt.Println("  - 文本 3 关于 Python，与前两个文本相似度较低")
	fmt.Println("  - 注意：这是简化的词频向量，真实 Embedding 模型效果更好")
}

// demoRetriever 演示检索器
func demoRetriever() {
	ctx := context.Background()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              Retriever 检索示例                             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	// 创建组件
	embedder := NewSimpleEmbedder(64)
	indexer := NewMemoryIndexer()
	retriever := NewMemoryRetriever(indexer, embedder)

	// 准备文档
	documents := []*Document{
		NewDocument("doc1", "Go 语言是 Google 开发的编程语言，特点是简洁高效，支持并发编程"),
		NewDocument("doc2", "Python 是一种解释型语言，广泛用于数据科学和机器学习"),
		NewDocument("doc3", "Eino 是 Go 语言的 LLM 应用开发框架，支持构建 RAG 和 Agent"),
		NewDocument("doc4", "RAG 技术通过检索外部知识来增强 LLM 的回答能力"),
		NewDocument("doc5", "Go 语言的 goroutine 提供了轻量级的并发支持"),
	}

	// 索引文档
	fmt.Println("\n[1] 索引文档:")
	texts := make([]string, len(documents))
	for i, doc := range documents {
		texts[i] = doc.Content
		fmt.Printf("  %s: %s\n", doc.ID, doc.Content)
	}

	vectors, _ := embedder.EmbedStrings(ctx, texts)
	indexer.Store(ctx, documents, vectors)
	fmt.Printf("\n  成功索引 %d 个文档\n", len(documents))

	// 测试检索
	queries := []string{
		"Go 语言有什么特点？",
		"什么是 RAG？",
		"数据科学用什么语言？",
	}

	for _, query := range queries {
		fmt.Printf("\n[2] 查询: %s\n", query)
		results, _ := retriever.Retrieve(ctx, query, 3)

		fmt.Println("  检索结果:")
		for i, doc := range results {
			fmt.Printf("    %d. [%s] (相关度: %.4f) %s\n",
				i+1, doc.ID, doc.Score(), doc.Content)
		}
	}
}

// demoChatTemplate 演示 ChatTemplate
func demoChatTemplate() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              ChatTemplate 提示词模板示例                     ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	// 示例 1：简单的 RAG 提示词模板
	fmt.Println("\n[示例 1] RAG 提示词模板:")
	template := &ChatTemplate{
		SystemPrompt: `你是一个智能助手。请基于以下参考资料回答问题。

参考资料：
{context}

要求：如果参考资料中没有相关信息，请如实说明。`,
		UserPrompt: "{question}",
	}

	variables := map[string]any{
		"context":  "Go 语言是 Google 开发的编程语言，特点是简洁高效。",
		"question": "Go 语言是谁开发的？",
	}

	messages := template.Format(variables)
	for _, msg := range messages {
		fmt.Printf("\n[%s]\n%s\n", msg.Role, msg.Content)
	}

	// 示例 2：多轮对话模板
	fmt.Println("\n\n[示例 2] 带历史对话的模板:")
	template2 := &ChatTemplate{
		SystemPrompt: "你是一个 helpful 的助手。请用中文回答问题。",
		UserPrompt:   "历史对话：{history}\n\n当前问题：{question}",
	}

	variables2 := map[string]any{
		"history":  "用户: 什么是 Go 语言？\n助手: Go 语言是 Google 开发的编程语言。",
		"question": "它有什么特点？",
	}

	messages2 := template2.Format(variables2)
	for _, msg := range messages2 {
		fmt.Printf("\n[%s]\n%s\n", msg.Role, msg.Content)
	}
}

// interactiveRAG 交互式 RAG 查询
func interactiveRAG() {
	ctx := context.Background()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              交互式 RAG 查询                                ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	// 创建 RAG 应用
	app := NewRAGApplication()

	// 预置知识库
	knowledgeBase := []string{
		"Go 语言（Golang）是 Google 开发的开源编程语言，特点是简洁、高效、支持并发。Go 语言使用 goroutine 实现轻量级线程，使用 channel 进行线程间通信。",
		"Python 是一种高级编程语言，语法简洁易读，广泛用于数据科学、机器学习、Web 开发等领域。Python 有丰富的第三方库，如 NumPy、Pandas、TensorFlow 等。",
		"Eino 是 CloudWeGo 团队开发的 Go 语言 AI 应用框架。Eino 提供了 ChatModel、Tool、Retriever、Embedding 等组件，支持构建 RAG 和 Agent 应用。",
		"RAG（检索增强生成）是一种 AI 技术，通过从知识库检索相关文档来增强 LLM 的回答。RAG 可以减少幻觉、提供最新信息、支持知识追溯。",
		"向量数据库是专门用于存储和检索向量的数据库。常见的向量数据库包括 Milvus、Pinecone、Weaviate、Chroma 等。向量数据库支持高效的相似性搜索。",
	}

	// 索引知识库
	fmt.Println("\n正在初始化知识库...")
	err := app.IndexDocuments(ctx, knowledgeBase)
	if err != nil {
		fmt.Printf("初始化失败: %v\n", err)
		return
	}
	fmt.Println("知识库初始化完成！")

	// 交互式查询
	fmt.Println("\n输入问题进行查询（输入 'quit' 退出）:")
	fmt.Println(strings.Repeat("-", 40))

	reader := os.Stdin
	buf := make([]byte, 1024)

	for {
		fmt.Print("\n请输入问题> ")

		n, err := reader.Read(buf)
		if err != nil {
			break
		}

		question := strings.TrimSpace(string(buf[:n]))
		if question == "" {
			continue
		}
		if question == "quit" || question == "exit" {
			fmt.Println("再见！")
			break
		}

		// 执行查询
		answer, docs, err := app.Query(ctx, question, 3)
		if err != nil {
			fmt.Printf("查询失败: %v\n", err)
			continue
		}

		// 显示结果
		fmt.Println("\n--- 检索结果 ---")
		for i, doc := range docs {
			fmt.Printf("[%d] (相关度: %.4f) %s\n", i+1, doc.Score(), doc.Content)
		}

		fmt.Println("\n--- 回答 ---")
		fmt.Println(answer)
	}
}

// showEinoRAGExample 展示真实 Eino RAG 代码结构
func showEinoRAGExample() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║          真实 Eino RAG 代码结构示例                         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	fmt.Println(`
以下是使用真实 Eino 框架构建 RAG 应用的代码结构：

// 1. 导入所需的包
import (
    "github.com/cloudwego/eino/components/document"
    "github.com/cloudwego/eino/components/embedding"
    "github.com/cloudwego/eino/components/indexer"
    "github.com/cloudwego/eino/components/retriever"
    "github.com/cloudwego/eino/components/prompt"
    "github.com/cloudwego/eino/schema"

    // 使用 eino-ext 中的具体实现
    openaiEmb "github.com/cloudwego/eino-ext/components/embedding/openai"
    milvusIdx "github.com/cloudwego/eino-ext/components/indexer/milvus"
    milvusRet "github.com/cloudwego/eino-ext/components/retriever/milvus"
)

// 2. 创建 Embedding 模型
emb, err := openaiEmb.NewEmbedder(ctx, &openaiEmb.EmbeddingConfig{
    Model: "text-embedding-3-small",
    APIKey: apiKey,
})

// 3. 创建 Indexer（向量存储）
idx, err := milvus.NewIndexer(ctx, &milvus.IndexerConfig{
    CollectionName: "my_knowledge_base",
    Embedding:      emb,
})

// 4. 加载和分割文档
loader := document.NewTextLoader(...)
docs, _ := loader.Load(ctx, document.Source{URI: "path/to/file.txt"})

splitter := document.NewSplitter(...)
chunks, _ := splitter.Transform(ctx, docs)

// 5. 索引文档
ids, _ := idx.Store(ctx, chunks)

// 6. 创建 Retriever
ret, _ := milvus.NewRetriever(ctx, &milvus.RetrieverConfig{
    CollectionName: "my_knowledge_base",
    Embedding:      emb,
    TopK:           5,
})

// 7. 检索文档
docs, _ := ret.Retrieve(ctx, "什么是 Eino？")

// 8. 构建提示词并生成回答
template := prompt.FromMessages(schema.FString,
    &schema.Message{
        Role:    schema.System,
        Content: "基于以下参考资料回答：{context}",
    },
    &schema.Message{
        Role:    schema.User,
        Content: "{question}",
    },
)

messages, _ := template.Format(ctx, map[string]any{
    "context":  formatDocs(docs),
    "question": "什么是 Eino？",
})

// 9. 调用 LLM 生成回答
chatModel, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    Model: "gpt-4o",
})
response, _ := chatModel.Generate(ctx, messages)
fmt.Println(response.Content)
`)
}

// printUsage 打印使用说明
func printUsage() {
	fmt.Println("Eino RAG（检索增强生成）示例程序")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  go run main.go demo         - 运行完整 RAG 流程演示（推荐首先运行）")
	fmt.Println("  go run main.go document     - 文档加载和分割示例")
	fmt.Println("  go run main.go embedding    - Embedding 向量嵌入示例")
	fmt.Println("  go run main.go retriever    - Retriever 检索示例")
	fmt.Println("  go run main.go template     - ChatTemplate 提示词模板示例")
	fmt.Println("  go run main.go rag          - 完整 RAG 应用示例")
	fmt.Println("  go run main.go interactive  - 交互式 RAG 查询")
	fmt.Println("  go run main.go eino         - 展示真实 Eino RAG 代码结构")
	fmt.Println("")
	fmt.Println("说明:")
	fmt.Println("  本示例使用模拟的 Embedding 和内存向量存储，无需外部依赖。")
	fmt.Println("  所有示例都可以直接运行，不需要 API Key。")
	fmt.Println("")
	fmt.Println("建议学习顺序:")
	fmt.Println("  1. demo      - 先了解 RAG 的完整流程")
	fmt.Println("  2. document  - 学习文档加载和分割")
	fmt.Println("  3. embedding - 学习向量嵌入")
	fmt.Println("  4. retriever - 学习检索器")
	fmt.Println("  5. template  - 学习提示词模板")
	fmt.Println("  6. rag       - 运行完整应用")
	fmt.Println("  7. eino      - 了解真实 Eino 代码结构")
}

// ============================================================================
// 程序入口
// ============================================================================

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "demo":
			demoRAGFlow()
		case "document":
			demoDocumentLoading()
		case "embedding":
			demoEmbedding()
		case "retriever":
			demoRetriever()
		case "template":
			demoChatTemplate()
		case "rag":
			demoRAGFlow()
		case "interactive":
			interactiveRAG()
		case "eino":
			showEinoRAGExample()
		default:
			fmt.Printf("未知命令: %s\n\n", os.Args[1])
			printUsage()
		}
	} else {
		printUsage()
	}
}
