# Redis Stack 向量库改造实录（2026-07-12）

> **阶段二里程碑**:在 `test/` 脚手架里,用 Redis Stack 完整实现"索引建立 + 向量检索",作为切到 Milvus 前的对照,深入体会两者差异。
> 关联:[阶段一产物 test/](../20260705/03-下一步行动计划.md)

---

## 一、完成的产物（test/ 目录）

| 文件 | 作用 |
|---|---|
| `test/indexer.go`(已有) | redis indexer,`HSET` 写入 3 个 chunk + 向量 |
| `test/index_schema.go`(新) | `CreateIndex()` —— `FT.CREATE` 建 RediSearch 向量索引 |
| `test/cmd/index/main.go`(新) | 一次性建索引 provisioning 入口 |
| `test/retriever.go`(新) | `NewRedisRetriever()` + 自定义 `DocumentConverter`(distance→Score) |
| `test/cmd/recall/main.go`(新) | KNN 召回入口 |

## 二、端到端链路验证

query `"ReAct Agent 是什么"` → 三个 chunk 按 COSINE 距离升序返回:

| chunk | content | score(COSINE 距离,越小越近) |
|---|---|---|
| `:test.md:3` | 三种 Agent 模式 / **ReAct Agent** / ... | **0.4637**(最小=最相似)✅ |
| `:test.md:2` | 核心概念 / Eino 标准组件接口 | 0.5899 |
| `:test.md:1` | # Eino 入门(纯标题) | 0.7344(最大=最不相似) |

语义排序正确,数值量级拉得开 → embedding 区分度够、整条 `query→embed→KNN→Document` 链路通。🔴

---

## 三、关键认知突破:Redis Stack vs Milvus 六大对比(面试硬通货)

### 1. 向量能力的本质 🔴
- **Redis**:KV 存储 **外挂 RediSearch 模块**(Redis Stack 才捆绑)。普通 Redis 连 `FT.CREATE` 都不认识。
- **Milvus**:本体就是向量库,索引引擎是第一等公民。
- **证据**:普通 Redis `MODULE LIST` 为空、`FT._LIST`/`FT.CREATE` 报 `unknown command`;换 `redis/redis-stack-server` 后 `MODULE LIST` 有 `search`+`ReJSON`。

### 2. 建索引 vs 写数据的顺序 🟡
- **Redis**:`FT.CREATE` 对已存在 key **回填**、对新 key 自动索引 → **顺序无关**。
- **Milvus**:必须**先建 collection schema 才能插入**,顺序反了插不进。
- **证据**:建索引时没跑任何写入,`FT.INFO` 的 `num_docs` 立刻 = 3(回填)。

### 3. Schema / 字段管理 🔴
- **Redis**:索引侧(`FT.CREATE`)、写入侧(indexer `EmbedKey`)、检索侧(`VectorField`)三处字段名 + 维度(1024)+ 度量(COSINE)**全靠人肉对齐,错一处静默失败**。
- **Milvus**:一处 `IndexerConfig.Fields` 定义搞定。

### 4. 协议细节 🟡
- **Redis**:go-redis 做 `FT.SEARCH` 必须 `Protocol: 2` + `UnstableResp3: true`,否则 RESP3 原始格式解析失败(空结果)。
- **Milvus**:无此坑。

### 5. 结果解析 / Score 🔴(本次踩最深的坑)
- **Redis**:默认 `defaultResultParser` **不写 Score**,`distance→Score` 要自己写 `DocumentConverter`。而且 **`ReturnFields`(=RETURN)↔ converter 里 `doc.Fields["…"]` 的 key 必须完全一致**,不一致会:content 变空 / score 变 0,**且不报错(静默失败)**——因为自定义 converter 取不到字段时 Go map 返回零值。
- **三处必须对齐**:`FT.CREATE 字段名 ↔ ReturnFields(RETURN) ↔ converter 的 doc.Fields[key]`。
- **Milvus**:`OutputFields` 列一次,SDK 托管映射进 Document,不写 converter、不会静默丢内容。

### 6. 持久化取向 🟡
- **Redis**:容器不挂 `-v` 数据卷时,`docker rm` 即丢数据(缓存基因)。
- **Milvus**:etcd + minio 卷持久(数据库基因)。

**一句总话术**:*"Redis 做向量库,是给 KV 存储外挂 RediSearch 模块,索引/RETURN/解析三处字段人肉对齐、还得手写 converter 接 distance→Score,错了静默失败;Milvus 是把向量索引作为第一等公民的专用引擎,schema/index/映射/持久化都受管。走一遍 Redis Stack,就知道 Milvus 那套 config 背后替你兜了多少手动对齐的坑。"*

---

## 四、源码锚点(复盘用)

**eino-ext redis indexer** `@v0.0.0-20260616080858`
- `indexer.go:98` `Store()` → `:126` `pipelineHSet`(**只 HSET,不建索引**)
- `utils.go:24` `vector2Bytes`(float64→float32→小端字节)→ 决定 `FT.CREATE` 的 `Type=FLOAT32`

**eino-ext redis retriever** `@v0.0.0-20260711013131`
- `retriever.go:168` KNN query `[KNN k @vector_content $vec AS distance]`
- `retriever.go:172-178` `Return` 完全由 `ReturnFields` 拼出;`SortBy distance Asc`;`WithScores:false`
- `defaultResultParser`(**只填 Content/DenseVector/MetaData,不碰 Score**)
- `RetrieverConfig.Client` 字段注释:`Protocol:2` + `UnstableResp3:true`
- `consts.go` `SortByDistanceAttributeName = "distance"`

**eino schema** `@v0.7.13`
- `document.go:72` `WithScore(score)` → 写入 `MetaData["_score"]`(是方法不是字段!)
- `document.go:84` `Score()` → 读 `MetaData["_score"]`

---

## 五、当前水平重估

| 维度 | 变化 |
|---|---|
| RAG 检索链路 | 从"能跑通索引侧" → "索引 + 检索**端到端**,能讲清向量库抽象层" |
| 向量库知识 | 新增 Redis Stack(RediSearch)实操 + Redis/Milvus 系统对比,面试 RAG 方向补齐一大块 |
| eino 组件抽象 | 亲手验证"两个实现满足同一 `indexer.Indexer`/`retriever.Retriever` 接口,但 config 表面不通用" |
| **待改进** | Go 工程基础(包路径=目录 vs 文件名、复合字面量逗号规则、方法 vs 字段);语法级 AI 依赖偏高,下个 topic 编译错先自己啃 15 分钟 |

**本 topic 评分:8 / 10**。概念接近满分,扣分全在 Go 工程基础 + AI 依赖度。

## 六、附加实验:HNSW vs FLAT 🟡

换 `HNSWOptions` 重建索引(`FT.INFO` 确认 `algorithm=HNSW`)后,召回分数与 FLAT **逐位相同**(`0.4637/0.5899/0.7344`)。原因:距离是 query↔doc 的固定数学量,索引只改"搜索策略"不改距离值;HNSW 虽是近似,但 N=3 时全覆盖 → 退化成精确。差异只在万级以上规模才显现(延迟↓、recall 可能 <100%)。**索引类型选的是 recall/延迟/内存权衡,不改度量本身。**

## 七、下一步

切换 Milvus,对照本文 6 点 + 深入其分布式特性(用户自部署)。
