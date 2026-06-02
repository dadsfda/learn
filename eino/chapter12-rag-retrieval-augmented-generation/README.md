# 第 12 章：RAG（Retrieval-Augmented Generation）—— 检索增强生成

## 学习目标

通过本章学习，你将掌握：

1. **RAG 核心概念**：理解什么是 RAG、为什么需要 RAG、RAG 解决了什么问题
2. **RAG 完整流程**：文档加载 → 文档分割 → 向量嵌入 → 向量存储 → 检索 → 上下文增强生成
3. **Eino RAG 组件**：Document Loader、Document Transformer、Embedding、Indexer、Retriever、ChatTemplate
4. **内存向量存储**：实现一个简单的内存向量存储，无需外部数据库即可学习 RAG
5. **完整 RAG 应用**：从文档加载到最终生成回答的全流程实现

## 前置知识

- 第 1 章：ChatModel 和 Message（消息构建）
- Go 语言基础（接口、结构体、切片、Map）
- Go 泛型基础（`[I, O any]` 语法）
- context.Context 使用

## 核心概念

### 1. RAG 是什么？

**RAG**（Retrieval-Augmented Generation，检索增强生成）是一种将**外部知识检索**与**大语言模型生成**相结合的技术。

简单来说，RAG 的工作方式是：

```
用户提问 → 从知识库中检索相关文档 → 将文档作为上下文发给 LLM → LLM 基于上下文生成回答
```

**生活中的类比**：

想象你是一个学生在开卷考试：
- **没有 RAG 的 LLM**：闭卷考试，只能靠记忆回答，可能会"编造"答案
- **有 RAG 的 LLM**：开卷考试，可以翻阅参考资料，回答更准确

### 2. 为什么需要 RAG？

大语言模型（LLM）虽然强大，但有几个固有的局限性：

| 问题 | 描述 | RAG 如何解决 |
|------|------|-------------|
| **知识过时** | LLM 的训练数据有截止日期，无法知道最新信息 | 通过检索实时更新的知识库获取最新信息 |
| **幻觉问题** | LLM 可能编造看似合理但错误的信息 | 基于检索到的真实文档生成回答，减少幻觉 |
| **领域知识不足** | LLM 缺乏特定企业或领域的专业知识 | 将企业文档、专业资料纳入知识库 |
| **无法引用来源** | LLM 的回答无法追溯到具体来源 | 检索到的文档可以作为引用来源 |
| **上下文长度限制** | 无法将所有知识都放入提示词中 | 只检索最相关的文档片段 |

### 3. RAG 完整流程

一个完整的 RAG 流程包含以下步骤：

```
┌─────────────────────────────────────────────────────────────────┐
│                      离线索引阶段（Indexing）                      │
│                                                                 │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐  │
│  │ 文档加载  │ →  │ 文档分割  │ →  │ 向量嵌入  │ →  │ 向量存储  │  │
│  │ (Loader) │    │(Splitter)│    │(Embedding)│    │ (Indexer)│  │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘  │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                      在线查询阶段（Querying）                      │
│                                                                 │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐  │
│  │ 用户提问  │ →  │ 向量嵌入  │ →  │ 相似检索  │ →  │ 上下文增强│  │
│  │ (Query)  │    │(Embedding)│    │(Retriever)│    │  生成     │  │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

#### 步骤详解：

**步骤 1：文档加载（Document Loading）**
- 从各种来源（文件、网页、数据库）加载原始文档
- Eino 接口：`document.Loader`

**步骤 2：文档分割（Document Splitting）**
- 将长文档分割成较小的片段（chunks）
- 原因：LLM 上下文有限，且检索需要精确匹配
- Eino 接口：`document.Transformer`

**步骤 3：向量嵌入（Embedding）**
- 将文本转换为数值向量（一组数字）
- 语义相似的文本，向量也相近
- Eino 接口：`embedding.Embedder`

**步骤 4：向量存储（Indexing）**
- 将文档向量存储到向量数据库中
- 支持高效的相似性搜索
- Eino 接口：`indexer.Indexer`

**步骤 5：检索（Retrieval）**
- 将用户问题也转换为向量
- 在向量数据库中找到最相似的文档片段
- Eino 接口：`retriever.Retriever`

**步骤 6：上下文增强生成（Augmented Generation）**
- 将检索到的文档片段和用户问题组合成提示词
- 发送给 LLM 生成最终回答
- Eino 工具：`prompt.DefaultChatTemplate`

### 4. Eino 中的 RAG 组件

#### 4.1 Document（文档）

`schema.Document` 是 Eino 中文档的基础类型：

```go
type Document struct {
    ID       string         `json:"id"`        // 文档唯一标识
    Content  string         `json:"content"`    // 文档内容
    MetaData map[string]any `json:"meta_data"`  // 元数据（来源、分数等）
}
```

Document 还提供了便捷的元数据访问方法：
- `WithScore(score)` / `Score()` - 设置/获取相关性分数
- `WithDenseVector(vector)` / `DenseVector()` - 设置/获取稠密向量
- `WithSparseVector(sparse)` / `SparseVector()` - 设置/获取稀疏向量

#### 4.2 Loader（文档加载器）

```go
type Loader interface {
    Load(ctx context.Context, src Source, opts ...LoaderOption) ([]*schema.Document, error)
}
```

`Loader` 负责从外部来源读取原始内容。`Source` 包含一个 `URI` 字段，可以是本地文件路径或远程 URL。

#### 4.3 Transformer（文档转换器）

```go
type Transformer interface {
    Transform(ctx context.Context, src []*schema.Document, opts ...TransformerOption) ([]*schema.Document, error)
}
```

`Transformer` 负责对文档进行转换操作，如分割、过滤、合并等。这是实现文档分块（chunking）的关键组件。

#### 4.4 Embedder（向量嵌入器）

```go
type Embedder interface {
    EmbedStrings(ctx context.Context, texts []string, opts ...Option) ([][]float64, error)
}
```

`Embedder` 将文本转换为向量表示。返回的 `[][]float64` 中，每个 `[]float64` 对应一个输入文本的向量。

**重要**：索引和检索时必须使用同一个 Embedding 模型，否则向量维度或语义空间不匹配，检索结果会不准确。

#### 4.5 Indexer（索引器）

```go
type Indexer interface {
    Store(ctx context.Context, docs []*schema.Document, opts ...Option) ([]string, error)
}
```

`Indexer` 将文档（及其向量）存储到后端存储系统中。返回的是存储后分配的文档 ID。

#### 4.6 Retriever（检索器）

```go
type Retriever interface {
    Retrieve(ctx context.Context, query string, opts ...Option) ([]*schema.Document, error)
}
```

`Retriever` 根据查询字符串从存储中检索最相关的文档。返回的文档按相关性排序（最相关的在前）。

常用选项：
- `retriever.WithTopK(n)` - 返回前 n 个结果
- `retriever.WithScoreThreshold(threshold)` - 只返回分数高于阈值的结果

#### 4.7 ChatTemplate（聊天模板）

```go
// 创建模板
template := prompt.FromMessages(schema.FString,
    &schema.Message{
        Role:    schema.System,
        Content: "你是一个基于以下参考资料回答问题的助手。\n\n参考资料：\n{context}",
    },
    &schema.Message{
        Role:    schema.User,
        Content: "{question}",
    },
)

// 格式化模板
messages, err := template.Format(ctx, map[string]any{
    "context":  "检索到的文档内容...",
    "question": "用户的问题",
})
```

`ChatTemplate` 使用模板语法（支持 FString、GoTemplate、Jinja2）构建消息列表，非常适合构建包含动态上下文的提示词。

### 5. 向量相似度计算

在 RAG 中，我们需要计算两个向量的相似度。最常用的方法是**余弦相似度**（Cosine Similarity）：

```
余弦相似度 = (A · B) / (|A| × |B|)
```

其中：
- `A · B` 是向量的点积
- `|A|` 和 `|B|` 是向量的模（长度）

余弦相似度的值范围是 [-1, 1]：
- **1**：完全相同的方向（最相似）
- **0**：正交（无关）
- **-1**：完全相反（最不相似）

## 代码示例

### 示例 1：内存向量存储实现

本示例实现了一个简单的内存向量存储，无需外部数据库即可学习 RAG 的核心原理。

```go
// MemoryVectorStore 内存向量存储
type MemoryVectorStore struct {
    documents  []*schema.Document  // 存储的文档
    vectors    [][]float64         // 对应的向量
}

// Store 存储文档和向量
func (s *MemoryVectorStore) Store(docs []*schema.Document, vectors [][]float64) {
    s.documents = append(s.documents, docs...)
    s.vectors = append(s.vectors, vectors...)
}

// Search 搜索最相似的文档
func (s *MemoryVectorStore) Search(queryVector []float64, topK int) []*schema.Document {
    // 计算所有文档与查询的相似度
    type docScore struct {
        doc   *schema.Document
        score float64
    }

    scores := make([]docScore, len(s.documents))
    for i, doc := range s.documents {
        scores[i] = docScore{
            doc:   doc,
            score: cosineSimilarity(queryVector, s.vectors[i]),
        }
    }

    // 按相似度排序
    sort.Slice(scores, func(i, j int) bool {
        return scores[i].score > scores[j].score
    })

    // 返回前 topK 个结果
    results := make([]*schema.Document, 0, topK)
    for i := 0; i < topK && i < len(scores); i++ {
        scores[i].doc.WithScore(scores[i].score)
        results = append(results, scores[i].doc)
    }
    return results
}
```

### 示例 2：文档加载和分割

```go
// SimpleTextSplitter 简单的文本分割器
type SimpleTextSplitter struct {
    ChunkSize    int  // 每个块的最大字符数
    ChunkOverlap int  // 块之间的重叠字符数
}

// Split 将文档分割成多个小块
func (s *SimpleTextSplitter) Split(doc *schema.Document) []*schema.Document {
    var chunks []*schema.Document
    content := doc.Content

    for i := 0; i < len(content); i += s.ChunkSize - s.ChunkOverlap {
        end := i + s.ChunkSize
        if end > len(content) {
            end = len(content)
        }

        chunk := &schema.Document{
            ID:      fmt.Sprintf("%s_chunk_%d", doc.ID, len(chunks)),
            Content: content[i:end],
            MetaData: map[string]any{
                "source":  doc.ID,
                "chunk_index": len(chunks),
            },
        }
        chunks = append(chunks, chunk)

        if end == len(content) {
            break
        }
    }
    return chunks
}
```

### 示例 3：模拟 Embedding

在实际项目中，你会使用 OpenAI、Cohere 等提供的 Embedding API。这里我们用简单的词频统计来模拟：

```go
// SimpleEmbedder 简单的向量嵌入器（用于学习演示）
// 实际项目中应使用 OpenAI Embedding、Cohere 等
type SimpleEmbedder struct {
    Dimension int  // 向量维度
}

// EmbedStrings 将文本转换为向量
func (e *SimpleEmbedder) EmbedStrings(texts []string) [][]float64 {
    vectors := make([][]float64, len(texts))
    for i, text := range texts {
        vectors[i] = e.textToVector(text)
    }
    return vectors
}

// textToVector 将文本转换为固定维度的向量
// 这里使用简化的哈希方法，实际项目中使用 Embedding 模型
func (e *SimpleEmbedder) textToVector(text string) []float64 {
    vector := make([]float64, e.Dimension)
    words := strings.Fields(strings.ToLower(text))

    for _, word := range words {
        hash := fnv.New32a()
        hash.Write([]byte(word))
        idx := int(hash.Sum32()) % e.Dimension
        vector[idx] += 1.0
    }

    // 归一化
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
```

### 示例 4：Retriever 实现

```go
// MemoryRetriever 基于内存向量存储的检索器
type MemoryRetriever struct {
    store     *MemoryVectorStore
    embedder  *SimpleEmbedder
    topK      int
}

// Retrieve 检索与查询最相关的文档
func (r *MemoryRetriever) Retrieve(ctx context.Context, query string) ([]*schema.Document, error) {
    // 1. 将查询转换为向量
    queryVectors := r.embedder.EmbedStrings([]string{query})
    if len(queryVectors) == 0 {
        return nil, fmt.Errorf("failed to embed query")
    }

    // 2. 搜索最相似的文档
    results := r.store.Search(queryVectors[0], r.topK)
    return results, nil
}
```

### 示例 5：ChatTemplate 构建带上下文的提示

```go
// buildRAGPrompt 构建 RAG 提示词
func buildRAGPrompt(ctx context.Context, question string, docs []*schema.Document) ([]*schema.Message, error) {
    // 1. 构建上下文文本
    var contextParts []string
    for i, doc := range docs {
        contextParts = append(contextParts,
            fmt.Sprintf("[参考资料 %d] (相关度: %.2f)\n%s",
                i+1, doc.Score(), doc.Content))
    }
    contextText := strings.Join(contextParts, "\n\n")

    // 2. 使用 ChatTemplate 构建消息
    template := prompt.FromMessages(schema.FString,
        &schema.Message{
            Role: schema.System,
            Content: `你是一个智能助手。请基于以下参考资料回答用户的问题。
如果参考资料中没有相关信息，请如实说明。

参考资料：
{context}`,
        },
        &schema.Message{
            Role:    schema.User,
            Content: "{question}",
        },
    )

    // 3. 格式化模板
    return template.Format(ctx, map[string]any{
        "context":  contextText,
        "question": question,
    })
}
```

### 示例 6：完整 RAG 应用

```go
// RAGApplication 完整的 RAG 应用
type RAGApplication struct {
    store    *MemoryVectorStore
    embedder *SimpleEmbedder
    retriever *MemoryRetriever
    chatModel model.ChatModel  // 可选，用于真实 LLM 调用
}

// NewRAGApplication 创建 RAG 应用
func NewRAGApplication(dimension int, topK int) *RAGApplication {
    store := &MemoryVectorStore{}
    embedder := &SimpleEmbedger{Dimension: dimension}
    retriever := &MemoryRetriever{
        store:    store,
        embedder: embedder,
        topK:     topK,
    }

    return &RAGApplication{
        store:     store,
        embedder:  embedder,
        retriever: retriever,
    }
}

// IndexDocuments 索引文档（离线阶段）
func (app *RAGApplication) IndexDocuments(ctx context.Context, docs []*schema.Document) error {
    // 1. 分割文档
    splitter := &SimpleTextSplitter{
        ChunkSize:    200,
        ChunkOverlap: 50,
    }

    var allChunks []*schema.Document
    for _, doc := range docs {
        chunks := splitter.Split(doc)
        allChunks = append(allChunks, chunks...)
    }

    // 2. 生成向量
    texts := make([]string, len(allChunks))
    for i, chunk := range allChunks {
        texts[i] = chunk.Content
    }
    vectors := app.embedder.EmbedStrings(texts)

    // 3. 存储到向量库
    app.store.Store(allChunks, vectors)
    return nil
}

// Query 查询（在线阶段）
func (app *RAGApplication) Query(ctx context.Context, question string) (string, error) {
    // 1. 检索相关文档
    docs, err := app.retriever.Retrieve(ctx, question)
    if err != nil {
        return "", err
    }

    // 2. 构建带上下文的提示
    messages, err := buildRAGPrompt(ctx, question, docs)
    if err != nil {
        return "", err
    }

    // 3. 如果有 ChatModel，调用 LLM 生成回答
    if app.chatModel != nil {
        resp, err := app.chatModel.Generate(ctx, messages)
        if err != nil {
            return "", err
        }
        return resp.Content, nil
    }

    // 4. 如果没有 LLM，返回检索结果摘要
    return buildFallbackAnswer(question, docs), nil
}
```

## 运行步骤

### 1. 环境准备

```bash
# 确保 Go 版本 >= 1.18
go version

# 进入章节目录
cd chapter12-rag-retrieval-augmented-generation
```

### 2. 运行示例

```bash
# 查看所有可用命令
go run main.go

# 运行基础 RAG 流程演示（不需要 API Key）
go run main.go demo

# 运行文档加载和分割示例
go run main.go document

# 运行 Embedding 示例
go run main.go embedding

# 运行 Retriever 检索示例
go run main.go retriever

# 运行 ChatTemplate 示例
go run main.go template

# 运行完整 RAG 应用示例
go run main.go rag

# 运行交互式 RAG 查询
go run main.go interactive
```

### 3. 使用真实 LLM（可选）

如果你想使用真实的 LLM（如 OpenAI）来生成最终回答：

```bash
# 设置 API Key
export OPENAI_API_KEY="your-api-key-here"

# 运行带 LLM 的完整 RAG
go run main.go rag-llm
```

## 常见问题

### Q1: 为什么需要文档分割？

**A**: 原因有三个：
1. **LLM 上下文限制**：LLM 的上下文窗口有限，无法处理超长文档
2. **检索精度**：较小的文档片段能更精确地匹配查询
3. **向量质量**：过长的文本生成的向量可能丢失细节信息

一般建议每个块（chunk）在 200-1000 个字符之间，具体取决于内容类型。

### Q2: Chunk Overlap（重叠）有什么用？

**A**: 重叠确保分割不会切断重要信息。例如，一个句子可能跨两个块，重叠可以保证这个句子在两个块中都完整出现。

### Q3: 如何选择 TopK（返回多少个结果）？

**A**: TopK 的选择取决于：
- **准确性需求**：TopK 越大，包含相关信息的概率越高
- **上下文窗口**：TopK 太大可能超出 LLM 的上下文限制
- **一般建议**：3-10 个，具体需要根据实际效果调优

### Q4: 模拟 Embedding 和真实 Embedding 有什么区别？

**A**:
- **模拟 Embedding**（本教程使用）：基于词频哈希，速度快但不理解语义。例如"快乐"和"高兴"的向量不相似
- **真实 Embedding**（如 OpenAI text-embedding-3-small）：基于深度学习，真正理解语义。"快乐"和"高兴"的向量会非常相似

### Q5: 如何在生产环境中使用 RAG？

**A**: 生产环境建议：
1. 使用真实的 Embedding 模型（OpenAI、Cohere、智谱等）
2. 使用专业的向量数据库（Milvus、Pinecone、Weaviate、Chroma）
3. 使用 Eino 的 `eino-ext` 扩展包中的具体实现
4. 考虑混合检索（向量检索 + 关键词检索）

### Q6: RAG 和微调（Fine-tuning）有什么区别？

**A**:
| 方面 | RAG | 微调 |
|------|-----|------|
| 原理 | 检索外部知识增强生成 | 修改模型参数学习新知识 |
| 成本 | 低（只需向量数据库） | 高（需要 GPU 和训练数据） |
| 更新 | 实时（更新知识库即可） | 需要重新训练 |
| 适用场景 | 知识密集型问答 | 特定任务/风格调整 |

## 练习题

### 练习 1：改进文本分割器

当前的 `SimpleTextSplitter` 按字符数分割，可能会切断句子。请实现一个 `SentenceSplitter`，按句子边界分割文档。

提示：
- 使用句号（。）、问号（？）、感叹号（！）作为分割点
- 保持每个块在指定的大小范围内
- 处理块之间的重叠

### 练习 2：实现 BM25 检索

除了向量检索，BM25 是另一种常用的检索方法。请实现一个简单的 BM25 检索器。

提示：
- BM25 基于词频（TF）和逆文档频率（IDF）
- 需要对文档进行分词
- 可以与向量检索结合使用（混合检索）

### 练习 3：添加文档来源追踪

修改 RAG 应用，使其在回答中引用具体的文档来源。

要求：
- 回答中包含引用标记，如 [1]、[2]
- 在回答末尾列出完整的引用来源
- 包含文档 ID 和相关度分数

### 练习 4：实现多轮对话 RAG

扩展示例，支持多轮对话的 RAG。每轮对话需要：
- 保持对话历史
- 根据对话上下文优化检索查询
- 在提示词中包含历史对话

### 练习 5：接入真实 Embedding

将模拟 Embedding 替换为真实的 Embedding API（如 OpenAI text-embedding-3-small）。

提示：
- 参考 `github.com/cloudwego/eino-ext` 中的 Embedding 实现
- 注意 API 调用频率限制
- 考虑缓存已生成的向量

## 参考资料

### 官方文档
- [Eino 官方仓库](https://github.com/cloudwego/eino)
- [Eino 扩展包](https://github.com/cloudwego/eino-ext)
- [CloudWeGo 文档](https://www.cloudwego.io/zh/docs/eino/)

### RAG 相关论文和文章
- [Original RAG Paper](https://arxiv.org/abs/2005.11401) - Lewis et al., 2020
- [Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks](https://arxiv.org/abs/2005.11401)

### 向量数据库
- [Milvus](https://milvus.io/) - 开源向量数据库
- [Chroma](https://www.trychroma.com/) - AI 原生嵌入数据库
- [Pinecone](https://www.pinecone.io/) - 托管向量数据库服务

### Embedding 模型
- [OpenAI Embeddings](https://platform.openai.com/docs/guides/embeddings)
- [Cohere Embed](https://docs.cohere.com/docs/embeddings)
- [智谱 Embedding](https://open.bigmodel.cn/dev/api#text_embedding)

## 下一步学习

完成本章后，建议继续学习：

1. **第 13 章**：Agent 工具调用 - 让 LLM 调用外部工具
2. **第 14 章**：Graph 编排 - 使用 Eino Graph 构建复杂的 RAG 工作流
3. **第 15 章**：生产部署 - 将 RAG 应用部署到生产环境
