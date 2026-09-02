package main

import (
	"context"
	"log"
	"os"

	"OncallAgent/test"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/compose"
	callbacksHelper "github.com/cloudwego/eino/utils/callbacks"
)

func main() {
	ctx := context.Background()

	// 1. 为每种组件创建专用 callback handler，观察每一步的输入输出
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
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			log.Printf("[Splitter 错误] %v", err)
			return ctx
		},
	}

	indexerHandler := &callbacksHelper.IndexerCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *indexer.CallbackInput) context.Context {
			log.Printf("[Indexer 开始] 准备写入 %d 个 Document", len(input.Docs))
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *indexer.CallbackOutput) context.Context {
			log.Printf("[Indexer 结束] 写入完成，IDs: %v", output.IDs)
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			log.Printf("[Indexer 错误] %v", err)
			return ctx
		},
	}

	embeddingHandler := &callbacksHelper.EmbeddingCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *embedding.CallbackInput) context.Context {
			log.Printf("[Embedding 开始] 向量化 %d 段文本", len(input.Texts))
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *embedding.CallbackOutput) context.Context {
			log.Printf("[Embedding 结束] 生成 %d 个向量（维度: %d）", len(output.Embeddings), len(output.Embeddings[0]))
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			log.Printf("[Embedding 错误] %v", err)
			return ctx
		},
	}

	allHandlers := callbacksHelper.NewHandlerHelper().
		Loader(loaderHandler).
		Transformer(transformerHandler).
		Indexer(indexerHandler).
		Embedding(embeddingHandler).
		Handler()

	// 2. 准备测试文档
	docPath := "./docs/test.md"
	content := "# Eino 入门\n## 核心概念\nEino 提供标准组件接口\n\n## 三种 Agent 模式\n- ReAct Agent\n- Plan-Execute-Replan\n- RAG Pipeline\n"
	if err := os.MkdirAll("./docs", 0755); err != nil {
		log.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(docPath, []byte(content), 0644); err != nil {
		log.Fatalf("write test doc failed: %v", err)
	}
	log.Printf("测试文档已创建: %s\n", docPath)

	// 3. 构建 Knowledge Indexing 管道
	runner, err := test.BuildKnowledgeIndexing(ctx)
	if err != nil {
		log.Fatalf("BuildKnowledgeIndexing failed: %v", err)
	}

	// 4. 运行管道
	source := document.Source{URI: docPath}
	result, err := runner.Invoke(ctx, source, compose.WithCallbacks(allHandlers))
	if err != nil {
		log.Fatalf("runner.Invoke failed: %v", err)
	}

	// 5. 打印结果
	log.Printf("✅ 索引完成！共 %d 个 chunk，IDs: %v\n", len(result), result)
}
