# Know-RAG · 企业级知识库 RAG 系统

> 一套面向企业私有知识场景、可生产落地的 **RAG（Retrieval‑Augmented Generation）** 系统。提供文档解析、结构化分块、向量检索、混合召回、查询改写、对话记忆、引用溯源、可观测追踪等完整能力。

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Vue](https://img.shields.io/badge/Vue-3.5-42B883?logo=vuedotjs&logoColor=white)](https://vuejs.org)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](#许可证)
[![Framework](https://img.shields.io/badge/go--zero-1.10-orange)](https://github.com/zeromicro/go-zero)
[![Vector DB](https://img.shields.io/badge/Milvus-2.6-00A1B7)](https://milvus.io)
[![Eino](https://img.shields.io/badge/Cloudwego-Eino-FF6F00)](https://github.com/cloudwego/eino)
[![RocketMQ](https://img.shields.io/badge/RocketMQ-5.3-D40000)](https://rocketmq.apache.org)
[![MinIO](https://img.shields.io/badge/MinIO-Storage-C72E29)](https://min.io)

---

## 目录

- [项目概述](#项目概述)
- [项目亮点](#项目亮点)
- [核心功能](#核心功能)
- [技术架构](#技术架构)
- [项目结构](#项目结构)
- [技术栈](#技术栈)
- [核心流程与执行模式](#核心流程与执行模式)
- [数据库与基础设施](#数据库与基础设施)
- [可观测与追踪](#可观测与追踪)
- [快速开始](#快速开始)

---

## 项目概述

**Know-RAG**是一套面向**企业私有知识库**场景的 **RAG** 服务，专注于解决以下工程问题：

- **结构化文档解析**：识别标题层级、章节、列表、表格、引用、附录等结构信号，构造文档骨架并自动生成最优分块策略。
- **高质量分块**：内置 `Recursive / Semantic / LLM` 等多种分块策略，自动按文档特征推荐，支持人工微调。
- **混合检索与重排**：向量召回 + 关键词召回 + RRF 融合 + 可选 Rerank，统一召回质量。
- **多模式对话**：`document / open_chat / auto_document` 三种模式，按需切换纯文档问答、自由对话或自动路由。
- **会话级记忆压缩**：基于 `summary_compression` 策略，自动生成结构化长期记忆，兼顾长上下文与成本。
- **在线回答评估**：内置 RAG 评估器（Faithfulness / Relevancy / Context Precision / Context Recall），在每轮对话的回答评估阶段对答案实时打分。
- **引用溯源与可观测**：每一次会话均产出 `DebugTrace` + `StageTrace` + `ChannelExecution` + `RetrievalResult`，可逐阶段回放。
- **异步流水线**：基于 RocketMQ 的「解析 → 索引构建」解耦，主链路响应毫秒级。

系统采用 **go-zero 微服务框架** + **领域驱动设计（DDD）分层**架构，模块边界清晰、易于二次开发。

---

## 项目亮点

1. **DDD + 手动注入的清晰架构**：领域 / 基础设施 / 触发器严格分层，依赖在 `cmd/bootstrap.go` 中按域手动组装，二次开发不污染核心。
2. **多策略分块自适应**：`Recursive / Semantic / LLM` 等分块策略按文档特征自动推荐，并支持人工微调。
3. **多模式路由与执行**：依据 `ChatQueryMode` 自动切换 `retrieval / clarification / 开放式自由对话` 等执行模式，覆盖从严格文档问答到开放式自由对话的全谱场景。
4. **混合检索 + RRF + Rerank**：向量召回 + 关键词召回统一打分，支持重排，可配置多种阈值（最低相似度、关键词相对分下限等）。
5. **结构化长期记忆**：`SummaryPayload` 含目标 / 事实 / 偏好 / 已解决 / 待办 / 检索提示六类，长会话仍能保持低成本高质量。
6. **完整可观测**：每一轮对话产出 `DebugTrace` + `StageTrace` + `ChannelExecution` + `RetrievalResult` + 评估快照，可逐阶段回放。
7. **异步流水线**：RocketMQ 解析 / 索引解耦，主链路响应毫秒级，错误可重试。
8. **可插拔基础设施**：Milvus / MinIO / MQ / LLM 全部在 `internal/infrastructure/port/` 内替换实现，迁移成本低。
9. **火山方舟深度集成**：Chat / Embedding / Rerank 同厂商，鉴权与限流策略一致。
10. **前后端分离**：后端 Go REST，前端 Vue 3 + Vite + Element Plus + Pinia。

---

## 核心功能

### 1. 文档管理
- 文档上传、解析、查询、删除
- 结构化骨架识别（章 / 节 / 小节 / 列表 / 表格 / 代码块 / 引用 / 附录）
- 文档画像（`DocumentProfile`）自动生成
- 异步分块策略推荐（人工可调整）
- 向量索引构建

### 2. 知识管理
- 知识范围（Scope）树形管理
- 知识主题（Topic）与文档的多对多关联
- 文档画像与主题路由
- 路由追踪（`KnowledgeRouteTrace`）

### 3. 智能问答
- 流式聊天（SSE）
- 多模式：`document` / `open_chat` / `auto_document`
- 查询改写（Query Rewrite + Sub-Question）
- 多通道检索（向量 + 关键词 + RRF 融合 + Rerank）
- 会话级长期记忆 + 上下文压缩
- 实时引用 + 章节级溯源
- 语义缓存（`SemanticCache`）—— 高相似问题直接复用历史答案
- 回答质量评估（Faithfulness / Relevancy / Context Precision / Context Recall）
- 追问推荐（`Recommendation`）
- 会话停止 / 重置 / 摘要重建

### 4. 检索与改写
- 向量检索：Milvus（基于 `eino-ext`）
- 关键词检索：Milvus 全文索引
- 文档导航（`DocumentQuestionRouter`）：自动定位到具体章节 / 条目
- 混合打分：RRF（Reciprocal Rank Fusion）+ Rerank（DashScope 可选）
- 检索管线：`ChannelRetrieval → Fusion → ParentElevation → Rerank → FinalTopK → Observation`

### 5. 执行模式（ExecutionMode）
对话由 `RouteStage` 依据 `ChatQueryMode` 与意图自动选择执行模式：

| 执行模式 | 触发条件 | 场景 |
| --- | --- | --- |
| `retrieval` | `document` / `auto_document` | 基于知识库的检索问答（默认） |
| `clarification` | 意图模糊 / 上下文不足 | 反问澄清，补全信息后再答 |
| 开放式自由对话 | `open_chat` | 不依赖文档的自由多轮对话 |

### 6. 记忆与上下文
- `summary_compression`：定时压缩历史，生成结构化 `SummaryPayload`（目标 / 事实 / 偏好 / 已解决 / 待办 / 检索提示）
- `sliding_window`：滑动窗口策略
- 自动覆盖轮次追踪、版本管理

---

## 技术架构

### 整体架构图

```
                ┌────────────────────────────────────────┐
                │            HTTP / SSE Client           │
                └──────────────────┬─────────────────────┘
                                   │
                ┌──────────────────▼─────────────────────┐
                │   go-zero REST API Layer (api/*)       │
                │  - chat    - document    - knowledge   │
                └──────────────────┬─────────────────────┘
                                   │
                ┌──────────────────▼─────────────────────┐
                │  Trigger Handler（HTTP/MQ 入口）        │
                │  - chat_service   - document_service   │
                │  - knowledge_service                   │
                └──────────────────┬─────────────────────┘
                                   │
                ┌──────────────────▼─────────────────────┐
                │       Domain Logic (DDD)               │
                │  ┌────────────┬────────────┬──────────┐│
                │  │  Chat      │ Document   │ Knowledge││
                │  │  • 改写     │ • 解析     │ • 路由    ││
                │  │  • 检索     │ • 分块     │ • 主题    ││
                │  │  • 记忆     │ • 索引     │ • 画像    ││
                │  │  • 推荐     │ • 策略     │           ││
                │  │  • 生成     │           │           ││
                │  └────────────┴────────────┴──────────┘│
                └──────────────────┬─────────────────────┘
                                   │
                ┌──────────────────▼─────────────────────┐
                │   Infrastructure Adapters              │
                │  MySQL · Redis · MinIO · Milvus · MQ   │
                │  LLM(Ark/Doubao) · Embedding · Rerank  │
                └────────────────────────────────────────┘

前端（know-agent/front）
                ┌────────────────────────────────────────┐
                │  Vue 3 + Vite + Element Plus + Pinia   │
                │  - ChatPage（用户对话）                │
                │  - AdminLayout / Dashboard             │
                │  - DocumentList / DocumentDetail       │
                │  - KnowledgeRoute / RouteTrace         │
                │  - ObservabilityList / Session / Detail│
                └────────────────────────────────────────┘
```

### 模块分层（DDD）
- `api/`：接口定义（`.api` 文件）+ HTTP 路由（go-zero 生成）
- `cmd/`：启动入口
- `internal/config`：配置加载
- `internal/domain/<aggregate>/`
  - `adapter/`：外部接口契约（port / repository）
  - `logic/`：业务实现
  - `model/entity|vo`：领域实体与值对象
  - `support/`：领域支撑工具
- `internal/infrastructure/`：基础设施实现（MySQL、Milvus、MinIO、Redis、RocketMQ、LLM 等）
- `internal/trigger/`：消费者 / 生产者适配入口
- `internal/server/`：服务组装
- `internal/svc/`：服务上下文
- `common/`：通用工具（Snowflake、JSON、条件、转换器等）

### 依赖手动注入
`cmd/bootstrap.go` 在启动期按域（chat / document / knowledge / infrastructure / server）手动组装完整服务图，无需代码生成，依赖关系直观、易于调试。

### 对话链路（Conversation Chain）
由 `conversation.NewChain` 装配的 13 阶段流水线：

```
Start → MemoryLoad → IntentRecognize → QueryRewrite
     → SemanticCache → Route → Retrieval → EvidenceBudget
     → Generate → CacheWrite → AnswerEvaluate → Recommend → End
```

其中 `AnswerEvaluate` 是**条件阶段**：仅在非开放闲聊、有检索结果且未命中语义缓存时执行；每个评估器独立 goroutine，并发产出快照。

---

## 项目结构

```
know-agent/
├── api/                              # 接口层（go-zero .api 定义）
│   ├── chat/                         #   - 聊天服务
│   ├── document/                     #   - 文档服务
│   └── knowledge/                    #   - 知识服务
├── cmd/                              # 启动入口
│   ├── main.go                       #   - 加载配置、启动 HTTP
│   ├── bootstrap.go                  #   - 手动依赖组装（含 13 阶段 Chain）
│   └── vector_retrieval_test.go      #   - 向量检索集成测试（连真实 Milvus）
├── common/                           # 通用工具
│   ├── base_config.go                #   - 配置基类
│   ├── biz_error.go                  #   - 业务错误
│   ├── json_array.go                 #   - JSON 数组辅助
│   ├── model.go                      #   - 通用模型
│   ├── response.go                   #   - 统一响应
│   └── utils/                        #   - 工具集（Snowflake、随机数、字符串等）
├── doc/                              # OpenAPI 文档
│   ├── chat.json
│   ├── document.json
│   └── knowledge.json
├── etc/                              # 配置文件
│   ├── config-dev.yaml               #   - 开发环境配置
│   ├── config-prod.yaml              #   - 生产环境配置
│   ├── milvus_schema.json            #   - Milvus 集合结构
│   └── schema.sql                    #   - MySQL DDL
├── internal/                         # 内部实现
│   ├── config/                       #   - 配置结构
│   ├── convert/                      #   - 类型转换器
│   ├── domain/                       #   - 领域层
│   │   ├── chat/                     #     - 聊天域
│   │   │   ├── logic/conversation/   #       - 13 阶段 Chain + 评估阶段
│   │   │   ├── logic/retrieval/      #       - 检索管线（Channel/Fusion/Rerank/...）
│   │   │   ├── logic/evaluate/       #       - 4 类评估器
│   │   │   ├── logic/rewrite/        #       - 查询改写
│   │   │   ├── logic/memory/         #       - 会话记忆 + 摘要压缩
│   │   │   ├── logic/intent/         #       - 意图识别
│   │   │   ├── logic/route/          #       - 文档路由
│   │   │   └── logic/recommend/      #       - 追问推荐
│   │   ├── document/                 #     - 文档域
│   │   │   ├── logic/process/        #       - 文档处理流水线
│   │   │   │   ├── parse/            #         - 解析
│   │   │   │   ├── analysis/         #         - 分析（Chain + Stage）
│   │   │   │   ├── chunk/            #         - 分块（recursive/semantic/llm）
│   │   │   │   └── index/            #         - 索引构建
│   │   │   └── model/entity/         #       - 领域实体
│   │   ├── knowledge/                #     - 知识域（Scope/Topic/Router/Rank）
│   │   ├── callbacks/                #     - 回调域（编排 / 追踪回调）
│   │   └── provider.go               #     - 域构造函数集合
│   ├── error/                        #   - 错误码
│   ├── infrastructure/               #   - 基础设施适配
│   │   ├── model/                    #     - ORM 模型
│   │   ├── persistence/              #     - MySQL 仓储
│   │   ├── observability/            #     - 追踪注册
│   │   └── port/                     #     - 外部端口
│   │       ├── vector/               #       - Milvus 向量
│   │       ├── keyword/              #       - Milvus 关键词
│   │       ├── storage/              #       - MinIO + ElasticSearch
│   │       ├── mq/                   #       - RocketMQ
│   │       ├── llm/                  #       - Ark / Doubao Chat Model
│   │       ├── emb/                  #       - Embedder（多模态）
│   │       ├── rerank/               #       - DashScope Rerank
│   │       ├── cache/                #       - 语义缓存
│   │       ├── lock/                 #       - Redis 分布式锁
│   │       ├── parser/               #       - Markdown / 文档解析
│   │       ├── gateway/              #       - 域间适配网关
│   │       ├── prompt/               #       - 提示词模板
│   │       ├── tokenize/             #       - GSE 分词
│   │       └── config/               #       - 本地配置
│   ├── server/                       #   - HTTP 服务
│   ├── svc/                          #   - 服务上下文
│   └── trigger/                      #   - 触发器（消费者）
│       ├── consumer/                 #     - MQ 消费者（parse / build_index）
│       └── handler/                  #     - 业务处理器
├── front/                            # 前端工程（Vue 3 + Vite）
│   ├── src/
│   │   ├── views/
│   │   │   ├── ChatPage.vue          #   - 用户对话主页
│   │   │   └── admin/                #   - 管理后台
│   │   │       ├── AdminLayoutView.vue
│   │   │       ├── AdminDashboardView.vue
│   │   │       ├── AdminDocumentListView.vue
│   │   │       ├── AdminDocumentDetailView.vue
│   │   │       ├── AdminKnowledgeRouteView.vue
│   │   │       ├── AdminKnowledgeRouteTraceView.vue
│   │   │       ├── AdminObservabilityListView.vue
│   │   │       ├── AdminObservabilitySessionView.vue
│   │   │       └── AdminObservabilityDetailView.vue
│   │   ├── components/
│   │   │   ├── chat/                 #   - ChatHeader / ChatInput / ChatSidebar
│   │   │   ├── admin/                #     FeedbackButtons / MessageItem / ...
│   │   │   └── common/
│   │   └── api/                      #   - 后端接口封装
│   └── package.json                  # - Vue 3.5 / Vite 6 / Element Plus / Pinia
├── Dockerfile                        # 多阶段构建镜像
├── docker-compose.yml                # 一键拉起 Milvus / MinIO / RocketMQ
├── go.mod / go.sum                   # 依赖
└── README.md                         # 本文档
```

---

## 技术栈

| 类别 | 选型 | 用途 |
| --- | --- | --- |
| 语言（后端） | Go 1.25 | 主语言 |
| HTTP 框架 | [go-zero](https://github.com/zeromicro/go-zero) 1.10 | RESTful API / 路由 / 配置 |
| LLM 编排 | [Cloudwego Eino](https://github.com/cloudwego/eino) 0.9 | Eino Graph |
| LLM 模型 | 火山方舟（豆包系列） | Chat / Embedding / Rerank |
| 向量数据库 | Milvus 2.6 / v3.0-beta | 向量 + 全文检索 |
| 关系数据库 | MySQL（GORM） | 业务元数据、对话、追踪 |
| 缓存 | Redis（redsync） | 分布式锁、语义缓存热点 |
| 对象存储 | MinIO | 原始文档、解析文本 |
| 消息队列 | Apache RocketMQ 5.3 | 异步解析 / 索引构建 |
| 全文检索 | ElasticSearch（gse 分词） | 知识路由侧召回 |
| 配置 | YAML（go-zero conf） | 多环境配置 |
| 语言（前端） | TypeScript 5.9 + Vue 3.5 | 单页应用 |
| 前端构建 | Vite 6 | 开发与构建 |
| 前端 UI | Element Plus 2.14 | 组件库 |
| 前端状态 | Pinia 2.3 + persistedstate | 状态管理 |
| 容器化 | Docker / docker-compose | 本地一键启动依赖 |

---

## 核心流程与执行模式

### 单轮对话主流程
```
用户问题
  │
  ▼
[1] StartStage ── 分布式锁 + 会话初始化
  │
  ▼
[2] MemoryLoadStage ── 历史摘要 / 近期转录 / 上下文
  │
  ▼
[3] IntentRecognizeStage ── 意图识别
  │
  ▼
[4] QueryRewriteStage ── 主问题 + Sub-Questions
  │
  ▼
[5] SemanticCacheStage ── 高相似问题直接复用答案
  │
  ▼
[6] RouteStage ── 依据 ChatQueryMode / 意图：
  │   - document / auto_document → retrieval（检索问答）
  │   - 意图不清 → clarification（反问澄清）
  │   - open_chat → 开放式自由对话
  │
  ▼
[7] RetrievalStage ── 多通道检索（Vector + Keyword）── RRF 融合 + Rerank
  │
  ▼
[8] EvidenceBudgetStage ── 证据预算裁剪
  │
  ▼
[9] GenerateStage ── 提示词组装 + LLM 流式生成
  │
  ▼
[10] CacheWriteStage ── 语义缓存写入
  │
  ▼
[11] AnswerEvaluateStage（条件）── Faithfulness / Relevancy / Context Precision / Recall
  │
  ▼
[12] RecommendStage ── 追问推荐
  │
  ▼
[13] EndStage ── 引用绑定 / 阶段追踪落库（TraceRecorder）
```

### 检索管线（Retrieval Pipeline）
```
ChannelRetrieval → Fusion(RRF) → ParentElevation → Rerank(DashScope)
                → FinalTopK → Observation
```
- `ChannelRetrieval`：并行执行向量 / 关键词通道
- `Fusion`：RRF 倒数排名融合
- `ParentElevation`：回溯到父块，补足上下文
- `Rerank`：可选的精排
- `FinalTopK`：按 TopK 截断
- `Observation`：落库通道执行 / 检索结果明细

### 异步处理
- `trigger/consumer/parse_document.go`：消费解析任务 → 抽取结构 → 写 MinIO → 落库
- `trigger/consumer/build_index.go`：消费索引任务 → 选分块策略 → Embedding → 写 Milvus

---

## 数据库与基础设施

### MySQL 关键表（`etc/schema.sql`）

| 表名 | 用途 |
| --- | --- |
| `chat_dialogue` | 会话记录 |
| `chat_exchange` | 对话轮次（含 `debug_trace_json`） |
| `chat_exchange_trace_stage` | 阶段追踪（按 `stageCode / stageOrder / stageLevel`） |
| `chat_channel_execution` | 通道执行记录（向量 / 关键词） |
| `chat_memory_summary` | 会话记忆摘要 |
| `chat_retrieval_result` | 检索结果明细 |
| `document_*` | 文档、策略、任务、结构节点、Profile |
| `knowledge_*` | Scope / Topic / 关系 / RouteTrace |

### Milvus（`etc/milvus_schema.json`）
- Collection：`document_chunk_collection`
- 主键：`chunk_id`
- 字段：`document_id / parent_block_id / section_path / structure_node_id / text / dense_vector / sparse_vector / chunk_no / parent_block_no`
- 距离度量：`COSINE`
- 同时启用稠密向量与全文（关键词）索引

### MinIO 桶布局
- `agent-document/`
  - `rag/document/`：原始文档
  - `rag/parsed-text/`：解析后纯文本
---

## 可观测与追踪

每一次对话都自动落库完整的可观测数据：

- `ChatDebugTrace`
  - 原始问题 / 改写问题 / 子问题列表
  - 文档导航决策（结构锚点 / 条目锚点）
  - 检索上下文、Prompt 模板、引用列表
  - 工具调用轨迹、模型用量与成本
- `ConversationTraceStage`
  - 按阶段（`stageCode / stageOrder / stageLevel`）记录耗时、状态、快照、错误
- `ChatChannelExecution`
  - 每个通道（向量 / 关键词）的召回 / 接受 / 最终选中等指标
- `ChatRetrievalResult`
  - 每条召回的原始分、RRF 分、Rerank 分、门控、选中原因
- **评估快照**（`AnswerEvaluateStage`）
  - 在 `StageOutput.Snapshot` 中记录每个评估器的实时分数

可通过 `/chat/exchange/detail` 一键拉取单轮全部调试信息，便于排查"为什么答非所问"。

---

## 快速开始

### 1. 启动基础设施
```bash
docker-compose up -d   # 拉起 Milvus / MinIO / RocketMQ / MySQL / Redis
```

### 2. 初始化数据库
```bash
mysql -h<host> -uroot -p < etc/schema.sql
```

### 3. 启动后端
```bash
go run cmd/main.go -f etc/config-dev.yaml
```

### 4. 启动前端
```bash
cd front
npm install
npm run dev
```

## 许可证

本项目基于 **Apache License 2.0** 开源，详见 [LICENSE](./LICENSE)。
