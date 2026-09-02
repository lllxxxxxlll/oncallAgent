package main

import (
	"context"
	"log"
	"os"

	"OncallAgent/test2"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	callbacksHelper "github.com/cloudwego/eino/utils/callbacks"
)

func main() {
	ctx := context.Background()

	// 1. Callback 观察 Loader 和 Splitter 的输入输出
	loaderHandler := &callbacksHelper.LoaderCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *document.LoaderCallbackInput) context.Context {
			log.Printf("[Loader 开始]  文件: %s", input.Source.URI)
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *document.LoaderCallbackOutput) context.Context {
			log.Printf("[Loader 结束]  加载到 %d 个 Document", len(output.Docs))
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			log.Printf("[Loader 错误]  %v", err)
			return ctx
		},
	}

	transformerHandler := &callbacksHelper.TransformerCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *document.TransformerCallbackInput) context.Context {
			log.Printf("[Splitter 开始] 输入 %d 个 Document", len(input.Input))
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *document.TransformerCallbackOutput) context.Context {
			log.Printf("[Splitter 结束] 切分为 %d 个 chunk", len(output.Output))
			for i, doc := range output.Output {
				log.Printf("  chunk[%d]: ID=%s, Content preview: %.60s...", i, doc.ID, doc.Content)
			}
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			log.Printf("[Splitter 错误] %v", err)
			return ctx
		},
	}

	allHandlers := callbacksHelper.NewHandlerHelper().
		Loader(loaderHandler).
		Transformer(transformerHandler).
		Handler()

	// 2. 准备测试文档（比上次更复杂，观察切分效果）
	docPath := "./docs/test2.md"
	content := `# Eino 框架学习笔记

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
`
	if err := os.MkdirAll("./docs", 0755); err != nil {
		log.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(docPath, []byte(content), 0644); err != nil {
		log.Fatalf("write test doc failed: %v", err)
	}
	log.Printf("测试文档已创建: %s\n", docPath)

	// 3. 构建 Loader → Splitter 管道
	runner, err := test2.Buildfile(ctx)
	if err != nil {
		log.Fatalf("Buildfile failed: %v", err)
	}

	// 4. 运行
	source := document.Source{URI: docPath}
	result, err := runner.Invoke(ctx, source, compose.WithCallbacks(allHandlers))
	if err != nil {
		log.Fatalf("runner.Invoke failed: %v", err)
	}
	m := result.(map[string]any)
	docs := m["docs"].([]*schema.Document)
	// 5. 打印结果
	log.Printf("✅ 管道完成！共切分为 %d 个 chunk", len(docs))
	for i, doc := range docs {
		log.Printf("  chunk[%d]: ID=%s", i, doc.ID)
		log.Printf("            H1=%v, H2=%v", doc.MetaData["H1"], doc.MetaData["H2"])
	}
}
