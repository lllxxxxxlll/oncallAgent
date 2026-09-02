# Eino 框架学习笔记

## Graph 类型校验

Eino 在 Compile 阶段用 checkAssignable 校验每条边的类型。

## 状态字典

Graph 内部的 Channel 系统本质是 map[string]any。

## 三种 Agent 模式

### ReAct Agent
LLM 自主决定调用工具的顺序。

### Plan-Execute-Replan
先规划再执行，动态调整计划。

### RAG Pipeline
检索增强生成的经典管道。
