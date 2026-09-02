# OncallAgent

基于 **Go + Eino** 的 AI 运维值班助手。面向告警处理场景,通过 RAG(检索增强生成)+ ReAct 工具调用,回答告警分析、知识库检索、日志查询等问题,并支持文件上传入库与 AI 运维报告生成。

## 技术栈

| 层 | 技术 |
|---|---|
| Web 框架 | GoFrame v2 |
| AI 编排 | Eino v0.7.13(字节开源 Go Agent 框架) |
| 对话模型 | DeepSeek V3(火山引擎 ARK,OpenAI 兼容接口) |
| 嵌入模型 | DashScope text-embedding-v4(阿里云,输出 2048 维) |
| 向量数据库 | Milvus(standalone,localhost:19530) |
| 工具协议 | MCP(Model Context Protocol,本地 SSE server) |
| 数据库 | MySQL(GORM,通过 mysql_crud 工具访问) |
| 可观测性 | Prometheus(告警查询,通过工具访问) |

## 架构:三套 Eino Agent

```
internal/ai/agent/
├── chat_pipeline/              # RAG + ReAct 对话 Agent(核心入口)
│   ├── orchestration.go        #   Graph 构建:并行 RAG + 对话模板 → ReAct
│   ├── flow.go                 #   ReAct Agent 定义与工具绑定
│   ├── retriever.go            #   向量检索器(转调 internal/ai/retriever)
│   ├── prompt.go               #   ChatTemplate 与 system prompt
│   ├── lambda_func.go          #   输入预处理 Lambda
│   └── types.go                #   UserMessage 类型
│
├── knowledge_index_pipeline/   # 知识索引管道(离线)
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

- **chat_pipeline**:对话入口。检索知识库 + 对话模型 + 工具调用(ReAct),支持流式(SSE)与非流式两种返回。
- **knowledge_index_pipeline**:离线管道,把文档切 chunk、向量化后写入 Milvus,由文件上传接口触发。
- **plan_execute_replan**:高级 Agent 模式,先规划、再执行、根据结果重规划,对应 `/api/ai_ops`。

## 目录结构

```
OncallAgent/
├── main.go                    # 入口,GoFrame 服务(监听 :6872)
├── api/                       # 接口定义(chat / upload / ai_ops)
├── internal/
│   ├── ai/                    # AI 核心
│   │   ├── agent/             #   三套 Agent(chat_pipeline / knowledge_index_pipeline / plan_execute_replan)
│   │   ├── models/            #   DeepSeek 对话模型封装(读 config)
│   │   ├── embedder/          #   DashScope 嵌入模型封装
│   │   ├── retriever/         #   Milvus 检索器
│   │   ├── loader/            #   文件加载器
│   │   └── tools/             #   工具实现(get_current_time / mysql_crud / query_internal_docs / query_log / query_metrics_alerts)
│   ├── controller/chat/       # HTTP 控制器
│   └── logic/                 # 业务逻辑(chat / sse)
├── manifest/
│   ├── config/                # config.yaml.example(模板,提交) + config.yaml(本地真实配置,gitignore)
│   └── docker/                # Dockerfile / docker-compose(Milvus 全家桶)
├── test/                      # 独立验证程序:Knowledge Indexing 管道(Redis Stack 向量库变体)
├── test2/                     # 独立验证程序:Eino Loader → Splitter 管道
├── SuperBizAgentFrontend/     # 前端(静态页面 + Node)
└── utility/                   # 通用工具(client / middleware / 日志回调等)
```

> `test/` 与 `test2/` 是独立的验证程序,不在主服务调用链内;`test/` 里是一套基于 Redis Stack(RediSearch)的索引/检索实现,主服务(`internal/`)实际使用 Milvus。

## 快速开始

### 1. 启动依赖

```bash
# Milvus 全家桶(etcd + minio + standalone + attu),首次启动会拉取镜像
cd manifest/docker && docker compose up -d
# 验证:http://localhost:9091/healthz 返回 OK
```

> ⚠️ **首次启动需先上传文档**:知识库为空(collection 未加载)时直接对话会因检索失败而报错。请先通过 `POST /api/upload` 上传一篇文档完成入库,再开始对话。详见[已知问题与待办](#已知问题与待办)。

另外两个可选依赖,按需启动:

- **MCP server**:`query_log` 工具依赖本地 SSE MCP server,监听 `http://localhost:3000/sse`(腾讯云本地部署方案,需腾讯云 key,不在本仓库内)。
- **MySQL**:`mysql_crud` 工具依赖,由模型在调用时传入 DSN。

### 2. 配置

复制模板并填入真实密钥(真实配置文件已被 `.gitignore` 忽略,不会提交):

```bash
cp manifest/config/config.yaml.example manifest/config/config.yaml
# 编辑 config.yaml,填入 ARK / DashScope key、Redis 等
```

### 3. 启动服务

```bash
go run main.go
# 监听 :6872
```

### 4. 接口

| 接口 | 方法 | 说明 |
|---|---|---|
| `/api/chat` | POST | 非流式对话 |
| `/api/chat_stream` | POST | 流式对话(SSE) |
| `/api/upload` | POST | 文件上传(multipart,入库知识库) |
| `/api/ai_ops` | POST | AI 运维报告(Plan-Execute-Replan) |

## 工具(Tool)清单与状态

ReAct Agent(`chat_pipeline/flow.go`)当前绑定 5 类工具,另有 1 个已实现但未接入:

| 工具 | 实现文件 | 依赖 | 状态 |
|---|---|---|---|
| `get_current_time` | `tools/get_current_time.go` | 无 | ✅ 可用,无外部依赖 |
| `query_internal_docs` | `tools/query_internal_docs.go` | Milvus + DashScope embedding | ⚠️ 依赖 Milvus;出错时 `log.Fatal` 会退出进程 |
| `query_log`(MCP) | `tools/query_log.go` | 本地 MCP server(`localhost:3000/sse`) | ⚠️ 启动强依赖;MCP server 未启动会导致 Agent 构建失败 |
| `mysql_crud` | `tools/mysql_crud.go` | MySQL(DSN 由模型传参) | ⚠️ 从 `os.Stdin` 读取确认,HTTP 场景下 stdin 不可用 |
| `query_prometheus_alerts` | `tools/query_metrics_alerts.go` | Prometheus | ❌ 已被开关短路,实际不查询 Prometheus |
| `newSearchTool`(duckduckgo) | `chat_pipeline/tools_node.go` | 外网搜索 | ⚙️ 已实现,但未在 `flow.go` 中绑定 |

## 配置项说明

配置通过 GoFrame `g.Cfg()` 读取,键名与 `manifest/config/config.yaml.example` 对应:

| 键 | 说明 |
|---|---|
| `ds_think_chat_model` | DeepSeek「思考」模型(ARK key / base_url / model) |
| `ds_quick_chat_model` | DeepSeek「快速」模型(ARK key / base_url / model) |
| `doubao_embedding_model` | 嵌入模型(DashScope key / base_url / model) |
| `file_dir` | 知识库文档目录(绝对路径) |
| `mcp_url` | 本地 MCP server 地址 |
| `redis` | Redis 向量库连接(仅 `test/` 验证程序使用) |

## 已知问题与待办

- **Milvus collection 冷启动未 load,空库检索失败(高优先级)**
  - 现象:全新 / 空库环境下,不先上传文档直接对话会因检索失败而报错。
  - 根因(三层叠加):
    1. `utility/client/client.go` 的 `NewMilvusClient` 创建 collection + AUTOINDEX 索引后,从不调用 `LoadCollection`;
    2. eino-ext 的 milvus retriever `loadCollection`(`utils.go` 的 `LoadStateNotLoad` 分支)只检查索引存在就 `return nil`,并未真正 load collection —— 对比 indexer 侧同名函数,后者真的调用了 `LoadCollection`;
    3. retriever 的 `Retrieve` 将「零检索结果」硬编码为错误(`no results found`),空库即失败。
  - 绕过方式:先走 `/api/upload` 上传文档 —— 该链路经 indexer,其 `loadCollection` 正确,会完成 load + 灌数据,之后对话即可正常检索。
- **Milvus 地址硬编码**:`utility/client/client.go` 中 `localhost:19530` 未走配置,与 `mcp_url`、模型 key 的配置化不一致。
- **Prometheus 工具被短路**:`tools/query_metrics_alerts.go` 在 `queryPrometheusAlerts()` 顶部直接 `return` 空结果,后续查询逻辑为不可达代码。
- **`log.Fatal` 滥用**:`query_internal_docs.go`、`mysql_crud.go` 等工具内部对错误调用 `log.Fatal`,在 HTTP 请求链路中会直接终止整个进程而非返回错误。
- **`mysql_crud` 的交互确认**:从 `os.Stdin` 读取 `y/n` 确认,在 Web 服务(无 stdin)场景下会阻塞或失效。
- **端口不一致**:`config.yaml.example` 中 `server.address` 为 `:8000`,但 `main.go` 通过 `SetPort(6872)` 覆盖;且 docker-compose 中 attu 已占用 8000。
- **system prompt 与工具未对齐**:`prompt.go` 中列出的 `SearchLog / DescribeAlarms` 等为腾讯云托管 MCP 的旧工具名,与当前本地 MCP server 返回的工具列表不一致。
- **`query_log` 启动强依赖**:`GetLogMcpTool()` 在 Agent 构建阶段同步连接 MCP server,server 不可用会阻断整个对话链路。
