# Know-Agent · 企业级知识库 RAG 智能体

> 一套面向企业私有知识场景、可生产落地的 **RAG + Agent** 一体化系统。提供文档解析、结构化分块、向量检索、混合召回、查询改写、对话记忆、ReAct Agent、引用溯源与可观测追踪等完整能力。

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](#许可证)
[![Framework](https://img.shields.io/badge/go--zero-1.10-orange)](https://github.com/zeromicro/go-zero)
[![Vector DB](https://img.shields.io/badge/Milvus-2.6-00A1B7)](https://milvus.io)
[![Eino](https://img.shields.io/badge/Cloudwego-Eino-FF6F00)](https://github.com/cloudwego/eino)
[![RocketMQ](https://img.shields.io/badge/RocketMQ-5.3-D40000)](https://rocketmq.apache.org)
[![MinIO](https://img.shields.io/badge/MinIO-Storage-C72E29)](https://min.io)

---

## 目录

- [项目概述](#项目概述)
- [核心功能](#核心功能)
- [技术架构](#技术架构)
- [项目结构](#项目结构)
- [技术栈](#技术栈)
- [核心流程与执行模式](#核心流程与执行模式)
- [数据库与基础设施](#数据库与基础设施)
- [可观测与追踪](#可观测与追踪)
- [项目亮点](#项目亮点)

---

## 项目概述

**Know-Agent**（模块名 `github.com/swiftbit/know-agent`）是一套面向**企业私有知识库**场景的 **RAG（Retrieval-Augmented Generation）+ Agent** 一体化服务，专注于解决以下工程问题：

- **结构化文档解析**：识别标题层级、章节、列表、表格、引用、附录等结构信号，构造文档骨架并自动生成最优分块策略。
- **高质量分块**：内置 `Recursive / Semantic / Structure / LLM` 四种分块策略，自动按文档特征推荐。
- **混合检索与重排**：向量召回 + 关键词召回 + RRF 融合 + 可选 Rerank，统一召回质量。
- **多模式对话**：`document / open_chat / auto_document` 三种模式，按需切换纯文档问答、自由对话或自动路由。
- **可执行多种 Executor**：`RagChat / GraphOnly / GraphThenEvidence / Clarification / ReActAgent` 等可插拔执行器。
- **会话级记忆压缩**：基于 `summary_compression` 策略，自动生成结构化长期记忆，兼顾长上下文与成本。
- **引用溯源与可观测**：每一次会话均产出 `DebugTrace` 与 `StageTrace`，可逐阶段回放检索/改写/回答过程。
- **异步流水线**：基于 RocketMQ 的「解析 → 索引构建」解耦，主链路响应毫秒级。

系统采用 **go-zero 微服务框架** + **领域驱动设计（DDD）分层** + **依赖注入（Google Wire）** 架构，模块边界清晰、易于二次开发。

---

## 核心功能

### 1. 文档管理（`/api/document`）
- 文档上传、解析、查询、删除
- 结构化骨架识别（章/节/小节/列表/表格/代码块/引用/附录）
- 文档画像（`DocumentProfile`）自动生成
- 异步分块策略推荐（人工可调整）
- 向量索引构建

### 2. 知识管理（`/api/knowledge`）
- 知识范围（Scope）树形管理
- 知识主题（Topic）与文档的多对多关联
- 文档画像与主题路由
- 路由追踪（`KnowledgeRouteTrace`）

### 3. 智能问答（`/api/chat`）
- 流式聊天（SSE）
- 多模式：`document` / `open_chat` / `auto_document`
- 查询改写（Query Rewrite + Sub-Question）
- 多通道检索（向量 + 关键词 + RRF 融合 + Rerank）
- 会话级长期记忆 + 上下文压缩
- 实时引用 + 章节级溯源
- 会话停止 / 重置 / 摘要重建

### 4. 检索与改写
- 向量检索：Milvus（基于 `eino-ext`）
- 关键词检索：Milvus 全文索引
- 文档导航（`DocumentQuestionRouter`）：自动定位到具体章节/条目
- 混合打分：RRF（Reciprocal Rank Fusion）+ Rerank（DashScope 可选）

### 5. 执行器（Executor）
| Executor | 模式 | 场景 |
| --- | --- | --- |
| `RagChatExecutor` | 通用 RAG | 默认文档问答 |
| `GraphOnlyExecutor` | 仅图查询 | 文档结构/章节导航 |
| `GraphThenEvidenceExecutor` | 图先验 + 证据 | 大纲型/章节型提问 |
| `ClarificationExecutor` | 反问澄清 | 意图模糊、上下文不足 |
| `ReActAgentExecutor` | ReAct Agent | 复杂多步推理/工具调用 |

### 6. 记忆与上下文
- `summary_compression`：定时压缩历史，生成结构化 `SummaryPayload`（目标/事实/偏好/已解决/待办/检索提示）
- `sliding_window`：滑动窗口策略
- 自动覆盖轮次追踪、版本管理

---

## 技术架构

### 整体架构图

```
                ┌────────────────────────────────────────┐
                │            HTTP / SSE Client          │
                └──────────────────┬─────────────────────┘
                                   │
                ┌──────────────────▼─────────────────────┐
                │   go-zero REST API Layer (api/*)      │
                │  - chat    - document    - knowledge  │
                └──────────────────┬─────────────────────┘
                                   │
                ┌──────────────────▼─────────────────────┐
                │  Trigger Handler（HTTP/MQ 入口）         │
                │  - chat_service   - document_service   │
                │  - knowledge_service                   │
                └──────────────────┬─────────────────────┘
                                   │
                ┌──────────────────▼─────────────────────┐
                │       Domain Logic (DDD)               │
                │  ┌────────────┬────────────┬──────────┐│
                │  │  Chat      │ Document   │ Knowledge││
                │  │  • 改写     │ • 解析     │ • 路由   ││
                │  │  • 检索     │ • 分块     │ • 主题   ││
                │  │  • 记忆     │ • 索引     │ • 画像   ││
                │  │  • 生成     │ • 策略     │          ││
                │  └────────────┴────────────┴──────────┘│
                └──────────────────┬─────────────────────┘
                                   │
                ┌──────────────────▼─────────────────────┐
                │   Infrastructure Adapters              │
                │  MySQL · Redis · MinIO · Milvus · MQ   │
                │  LLM(Ark/Doubao) · Embedding · Rerank  │
                └────────────────────────────────────────┘
```

### 模块分层（DDD）
- `api/`：接口定义（`.api` 文件）+ HTTP 路由（go-zero 生成）
- `internal/config`：配置加载
- `internal/domain/<aggregate>/`
  - `adapter/`：外部接口契约（port / repository）
  - `logic/`：业务用例
  - `model/entity|vo`：领域实体与值对象
  - `support/`：领域支撑工具
- `internal/infrastructure/`：基础设施实现（MySQL、Milvus、MinIO、Redis、RocketMQ、LLM 等）
- `internal/trigger/`：消费者/生产者适配入口
- `internal/server/`：服务组装
- `internal/svc/`：服务上下文
- `internal/provider.go` + `cmd/wire.go` + `cmd/wire_gen.go`：Google Wire 依赖注入
- `common/`：通用工具（Snowflake、JSON、条件、转换器等）

### 依赖注入（Wire）
`cmd/wire.go` 通过 Google Wire 生成 `wire_gen.go`，运行时构造完整服务图。Provider 按域（chat / document / knowledge / infrastructure / server）拆分，边界清晰。

---

## 项目结构

```
know-agent/
├── api/                           # 接口层（go-zero .api 定义）
│   ├── chat/                      #   - 聊天服务
│   ├── document/                  #   - 文档服务
│   └── knowledge/                 #   - 知识服务
├── cmd/                           # 启动入口
│   ├── main.go                    #   - 加载配置、启动 HTTP
│   ├── wire.go                    #   - Wire 注入声明
│   └── wire_gen.go                #   - Wire 生成代码
├── common/                        # 通用工具
│   ├── base_config.go             #   - 配置基类
│   ├── biz_error.go               #   - 业务错误
│   ├── json_array.go              #   - JSON 数组辅助
│   ├── model.go                   #   - 通用模型
│   ├── response.go                #   - 统一响应
│   └── utils/                     #   - 工具集（Snowflake、随机数、字符串等）
├── doc/                           # OpenAPI 文档
│   ├── chat.json
│   ├── document.json
│   └── knowledge.json
├── etc/                           # 配置文件
│   ├── config-dev.yaml            #   - 开发环境配置
│   ├── milvus_schema.json         #   - Milvus 集合结构
│   └── schema.sql                 #   - MySQL DDL
├── internal/                      # 内部实现
│   ├── config/                    #   - 配置结构
│   ├── convert/                   #   - Wire 类型转换器
│   ├── domain/                    #   - 领域层
│   │   ├── chat/                  #     - 聊天域（改写/检索/记忆/Agent）
│   │   ├── document/              #     - 文档域（解析/分块/索引/策略）
│   │   ├── knowledge/             #     - 知识域（范围/主题/路由）
│   │   └── provider.go            #     - 域 Provider
│   ├── error/                     #   - 错误码
│   ├── infrastructure/            #   - 基础设施适配
│   │   ├── model/                 #     - ORM 模型
│   │   ├── persistence/           #     - MySQL 仓储
│   │   └── port/                  #     - 外部端口（Milvus/MinIO/MQ/LLM）
│   ├── server/                    #   - HTTP 服务
│   ├── svc/                       #   - 服务上下文
│   └── trigger/                   #   - 触发器（消费者）
│       ├── consumer/              #     - MQ 消费者
│       └── handler/               #     - 业务处理器
├── Dockerfile                     # 多阶段构建镜像
├── docker-compose.yml             # 一键拉起 Milvus / MinIO / RocketMQ
├── go.mod / go.sum                # 依赖
└── README.md                      # 本文档
```

---

## 技术栈

| 类别 | 选型 | 用途 |
| --- | --- | --- |
| 语言 | Go 1.26 | 主语言 |
| HTTP 框架 | [go-zero](https://github.com/zeromicro/go-zero) 1.10 | RESTful API / 路由 / 配置 |
| LLM 编排 | [Cloudwego Eino](https://github.com/cloudwego/eino) 0.9 | Eino Graph / ReAct Agent |
| LLM 模型 | 火山方舟（豆包系列） | Chat / Embedding / Rerank |
| 向量数据库 | Milvus 2.6 / v3.0-beta | 向量 + 全文检索 |
| 关系数据库 | MySQL（GORM） | 业务元数据、对话、追踪 |
| 缓存 | Redis（redsync） | 分布式锁、热点缓存 |
| 对象存储 | MinIO | 原始文档、解析文本 |
| 消息队列 | Apache RocketMQ 5.3 | 异步解析 / 索引构建 |
| 依赖注入 | Google Wire | 启动期依赖图生成 |
| 配置 | YAML（go-zero conf） | 多环境配置 |
| 容器化 | Docker / docker-compose | 本地一键启动依赖 |

---

## 核心流程与执行模式

### 单轮对话主流程
```
用户问题
  │
  ▼
[1] 记忆准备（SummaryCompression）── 历史摘要 / 近期转录 / 上下文
  │
  ▼
[2] 问题改写（QueryRewrite）── 主问题 + Sub-Questions
  │
  ▼
[3] 文档导航（DocumentQuestionRouter）── 结构/条目锚点
  │
  ▼
[4] 执行器选择（ExecutorRegistry）── 模式匹配：
  │   - RagChatExecutor
  │   - GraphOnlyExecutor
  │   - GraphThenEvidenceExecutor
  │   - ClarificationExecutor
  │   - ReActAgentExecutor
  │
  ▼
[5] 多通道检索（Vector + Keyword）── RRF 融合 + Rerank
  │
  ▼
[6] 提示词组装（RagPromptAssembler）── 模板 + 证据 + 历史
  │
  ▼
[7] LLM 生成（Ark / Eino）── 流式输出
  │
  ▼
[8] 引用绑定 / 阶段追踪落库（TraceRecorder）
```

### 异步处理
- `trigger/consumer/parse_document.go`：消费解析任务 → 抽取结构 → 写 MinIO → 落库
- `trigger/consumer/build_index.go`：消费索引任务 → 选分块策略 → Embedding → 写 Milvus

---

## 数据库与基础设施

### MySQL 关键表（`etc/schema.sql`）
- `chat_dialogue`：会话记录
- `chat_exchange`：对话轮次
- `chat_exchange_trace_stage`：阶段追踪
- `chat_channel_execution`：通道执行记录
- `chat_memory_summary`：会话记忆摘要
- `chat_retrieval_result`：检索结果明细
- `document_*`：文档、策略、任务、结构节点、Profile
- `knowledge_*`：Scope/Topic/关系/RouteTrace

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
  - 每个通道（向量/关键词）的召回/接受/最终选中等指标
- `ChatRetrievalResult`
  - 每条召回的原始分、RRF 分、Rerank 分、门控、选中原因

可通过 `/chat/exchange/detail` 一键拉取单轮全部调试信息，便于排查"为什么答非所问"。

---

## 项目亮点

1. **DDD + Wire 的清晰架构**：领域 / 基础设施 / 触发器严格分层，Provider 按域聚合，二次开发不污染核心。
2. **多策略分块自适应**：`Recursive / Semantic / Structure / LLM` 四种分块可按文档特征自动推荐，并支持人工微调。
3. **可执行多种 Executor**：`GraphOnly / GraphThenEvidence / Clarification / ReActAgent` 覆盖从纯结构查询到复杂推理的全谱场景。
4. **混合检索 + RRF + Rerank**：向量召回 + 关键词召回统一打分，支持重排，可配置多种阈值（最低相似度、关键词相对分下限等）。
5. **结构化长期记忆**：`SummaryPayload` 含目标/事实/偏好/已解决/待办/检索提示六类，长会话仍能保持低成本高质量。
6. **完整可观测**：每一轮对话产出 `DebugTrace` + `StageTrace` + `ChannelExecution` + `RetrievalResult`，可逐阶段回放。
7. **异步流水线**：RocketMQ 解析/索引解耦，主链路响应毫秒级，错误可重试。
8. **可插拔基础设施**：Milvus / MinIO / MQ / LLM 全部在 `internal/infrastructure/port/` 内替换实现，迁移成本低。
9. **火山方舟深度集成**：Chat / Embedding / Rerank 同厂商，鉴权与限流策略一致。
10. **多模式路由**：`document / open_chat / auto_document` 三种模式，覆盖"严格基于文档 / 自由对话 / 自动判断"。

---
