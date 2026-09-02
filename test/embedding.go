package test

import (
	"context"
	"os"

	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
	"github.com/cloudwego/eino/components/embedding"
)

func newEmbedding(ctx context.Context) (eb embedding.Embedder, err error) {
	config := &dashscope.EmbeddingConfig{
		APIKey: os.Getenv("DASHSCOPE_API_KEY"),
		Model:  "text-embedding-v4",
	}
	eb, err = dashscope.NewEmbedder(ctx, config)
	if err != nil {
		return nil, err
	}
	return eb, nil
}
