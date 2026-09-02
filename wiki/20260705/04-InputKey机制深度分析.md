# InputKey/OutputKey 机制深度分析

> 关键词：状态字典、Channel 系统、泛型 vs Key 模式、Compile 类型检查
> 面试重要度：🔴 必问
> 探索日期：2026-07-02 ~ 2026-07-05

---

## 一、状态字典的本质

Eino Graph 内部不维护一个全局 `map[string]any`，而是为**每个节点创建独立的 Channel**：

```go
// graph_manager.go
type dagChannel struct {
    Values              map[string]any  // key=前驱节点名, value=前驱输出
    ControlPredecessors map[string]dependencyState
    DataPredecessors    map[string]bool
}
```

当一个节点执行完，输出写入下游节点的 Channel：

```go
// graph_run.go:832
writeChannelValues[next][t.nodeKey] = vs[i]
// e.g. writeChannelValues["Splitter"]["FileLoader"] = docs
```

**Channel 内部 key = 前驱节点名（如 "start", "FileLoader"），和用户设置的 InputKey/OutputKey 是两套完全不同的体系。**

---

## 二、InputKey/OutputKey 是什么

它们不是类型约束，是**数据格式转换指令**。

### OutputKey

```go
// runnable.go:483
func outputKeyedComposableRunnable(key string, r *composableRunnable) *composableRunnable {
    wrapper.i = func(ctx, input any, ...) (any, error) {
        out, _ := i(ctx, input, ...)               // 执行组件，得到裸值
        return map[string]any{key: out}, nil        // 包成 map
    }
}
```

### InputKey

```go
// runnable.go:448
func inputKeyedComposableRunnable(key string, r *composableRunnable) *composableRunnable {
    wrapper.i = func(ctx, input any, ...) (any, error) {
        v := input.(map[string]any)[key]            // 从 map 按 key 取裸值
        return i(ctx, v, ...)                       // 传给组件
    }
}
```

### 对类型系统的影响

```go
// graph_node.go:93
func (gn *graphNode) inputType() reflect.Type {
    if len(gn.nodeInfo.inputKey) != 0 {
        return generic.TypeOf[map[string]any]()  // ← Key 模式：放弃类型信息
    }
    return gn.cr.inputType                       // ← 泛型模式：保留真实类型
}
```

---

## 三、泛型模式 vs Key 模式

```
泛型模式 (NewGraph[document.Source, []string]):
  组件输出裸值 → 直接传给下游 → Compile 时 checkAssignable 校验 ✅
  
Key 模式 (NewGraph[any, any] + InputKey/OutputKey):
  组件输出裸值 → outputKeyedRunnable 包成 map → 传给下游 
  → inputKeyedRunnable 从 map 取出 → 传给组件
  Compile 时只能检查 map[string]any ↔ map[string]any ✅（虚的）
```

### 关键差异

| | 泛型模式 | Key 模式 |
|---|---|---|
| 类型检查 | 编译期（Go 泛型） | 运行时（map key 匹配） |
| 数据传递 | 裸值直达 | 包 map / 解包 map |
| START 兼容 | ✅ 天然兼容 | ❌ START 不包 map，第一个节点不能设 InputKey |
| 调用方体验 | `result` 直接是目标类型 | `result` 是 map，还要解包 |

---

## 四、关键发现

### START 节点是特殊的

START 不经过 `outputKeyedComposableRunnable`，输出永远是裸值。所以 Key 模式下第一个节点不能设 InputKey，否则出现：

```
START 输出裸值 → 节点期望 map[string]any → 类型断言失败 panic
```

### 单前驱 vs 多前驱

```go
// dag.go:178
if len(valueList) == 1 {
    return valueList[0], true, nil  // 单前驱：返回裸值
}
v, _ := mergeValues(valueList, ...) // 多前驱：合并 map
```

多前驱汇聚时 `mergeValues` 合并多个 `map[string]any` 的 key，所以各分支用不同 OutputKey 是可行的，汇聚节点可以直接从合并后的 map 取值。

### Key 模式的合法场景

1. `NewGraph[any, map[string]any]()` — 通用 map 管道（低代码编排器）
2. 多前驱汇聚（fan-in）：各分支设 OutputKey，汇聚节点接收合并 map
3. 全局状态管理：`WithGenLocalState`
4. **不适用于标准类型流水线**（你的项目）

---

## 五、探索过程记录

1. 脚手架代码使用 `NewGraph[any, any]` + 手动 Key → 运行时报 `map[string]any` vs `document.Source` 类型错误
2. 对比生产代码发现使用 `NewGraph[document.Source, []string]` + 不设 Key
3. 阅读 `graph_node.go` 发现 InputKey 改变了 `inputType()` 返回值
4. 阅读 `runnable.go` 发现 outputKeyed/inputKeyed 的包装逻辑
5. 阅读 `dag.go` 发现单前驱 `get()` 返回裸值的逻辑
6. 阅读 `graph.go` 发现 Compile 时的 `checkAssignable` 类型校验
7. 通过 test2 验证了"第一个节点不设 InputKey 能跑通"的假设
8. 通过 fanin demo 验证了多前驱汇聚时 Key 的正确用法

## 六、相关源码位置

| 文件 | 关键内容 |
|---|---|
| `compose/graph.go` | `checkAssignable`, `getNodeInputType`, `getNodeOutputType` |
| `compose/graph_node.go` | `inputType()`, `compileIfNeeded()` |
| `compose/runnable.go` | `inputKeyedComposableRunnable`, `outputKeyedComposableRunnable` |
| `compose/dag.go` | `dagChannel`, `get()`, `reportValues()` |
| `compose/graph_manager.go` | `channelManager`, `updateAndGet()` |
| `compose/graph_run.go` | `resolveCompletedTasks`, `calculateNextTasks` |
| `compose/field_mapping.go` | `FieldMapping`, `mergeValues` |
| `compose/values_merge.go` | `mergeValues` |
