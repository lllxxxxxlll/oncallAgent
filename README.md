# OncallAgent

一个基于 **Go + Eino** 的 AI Agent 值班助手。面向 AI 运维场景,通过 RAG(检索增强生成)与 ReAct 工具调用,回答告警处理、知识库检索等问题。

> 本项目同时是作者系统学习 AI Agent 全链路技术栈的实战载体,当前处于开发/学习阶段。

## 技术栈

| 层 | 技术 |
|---|---|
| Web 框架 | GoFrame v2 |
| AI 编排 | Eino v0.7.13(字节开源 Go Agent 框架) |
| 对话模型 | DeepSeek V3(火山引擎 ARK,OpenAI 兼容接口) |
| 嵌入模型 | DashScope text-embedding-v4(阿里云) |
| 向量数据库 | Milvus / Redis Stack(RediSearch) |
| 工具协议 | MCP(Model Context Protocol) |
| 数据库 | MySQL(GORM) |
| 可观测性 | Prometheus(告警查询) |

## 架构:三套 Eino Agent

```
internal/ai/agent/
├── chat_pipeline/              # RAG + ReAct 对话 Agent
│   ├── orchestration.go        #   Graph 构建(并行 RAG + 对话)
│   ├── flow.go                 #   ReAct Agent Lambda
│   ├── retriever.go            #   向量检索器
│   ├── prompt.go               #   ChatTemplate 定义
│   └── types.go                #   UserMessage 类型
│
├── knowledge_index_pipeline/   # 知识索引管道
│   ├── orchestration.go        #   Graph:Loader → Splitter → Indexer
│   ├── loader.go               #   文件加载
│   ├── transformer.go          #   Markdown 切分
│   └── indexer.go              #   向量入库
│
└── plan_execute_replan/        # Plan-Execute-Replan Agent(ADK)
    ├── plan_execute_replan.go  #   ADK Runner 编排
    ├── planner.go              #   Planner(Think 模型)
    ├── executor.go             #   Executor(Quick 模型)
    └── replan.go               #   Replanner
```

- **chat_pipeline**:核心对话入口,检索知识库 + 对话模型 + 工具调用(ReAct)。
- **knowledge_index_pipeline**:离线管道,把文档切成 chunk、向量化后写入向量库。
- **plan_execute_replan**:高级 Agent 模式,先规划、再执行、根据结果重规划。

## 目录结构

```
OncallAgent/
├── main.go                    # 入口,GoFrame 服务(端口 6872)
├── api/                       # 接口定义(chat / upload / ai_ops)
├── internal/
│   ├── ai/                    # AI 核心:agent / models / embedder / loader / indexer / retriever / tools
│   ├── controller/chat/       # HTTP 控制器
│   └── logic/                 # 业务逻辑(chat / sse)
├── manifest/
│   ├── config/config.yaml     # 服务配置 + 模型 key
│   └── docker/                # Dockerfile / docker-compose(Milvus、Redis 等)
├── test/                      # 阶段一:Knowledge Indexing 管道(Redis 向量库,端到端跑通)
├── test2/                     # P0-1 验证:any + Key 模式的 Loader → Splitter 管道
├── SuperBizAgentFrontend/     # 前端(静态页面 + Node)
├── utility/                   # 通用工具(client / middleware / 日志回调等)
└── wiki/                      # 学习笔记与进度沉淀
```

## 快速开始

1. **启动依赖**(向量库等,二选一):

   ```bash
   # Milvus(项目自带 docker-compose)
   cd manifest/docker && ./docker.sh

   # 或 Redis Stack(RediSearch)
   docker run -d --name redis-stack -p 6379:6379 -p 8001:8001 redis/redis-stack-server
   ```

2. **配置模型**:编辑 `manifest/config/config.yaml`,填入 DeepSeek(ARK)与 DashScope 的 key。

3. **启动服务**:

   ```bash
   go run main.go
   # 监听 :6872
   ```

4. **接口**:

   | 接口 | 说明 |
   |---|---|
   | `POST /api/chat` | 普通对话 |
   | `POST /api/chat_stream` | 流式对话(SSE) |
   | `POST /api/upload` | 文件上传(入库知识库) |
   | `POST /api/ai_ops` | AI 运维 |

## 当前进度

- ✅ 阶段一:Knowledge Indexing 管道跑通(Loader → Splitter → Embedding → 向量入库)
- ✅ 阶段二:Redis Stack 向量库改造(FT.CREATE 建索引 + 自定义 DocumentConverter 召回,端到端跑通)
- ✅ P0-1:Eino InputKey/OutputKey 机制攻克
- ⏳ 进行中:切换 Milvus 对照体会、ReAct 对话链路、MCP 工具接入

详见 `wiki/` 目录下的学习笔记与进度评估。
