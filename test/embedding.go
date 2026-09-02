package test

import (
	"context"

	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/gogf/gf/v2/frame/g"
)

func newEmbedding(ctx context.Context) (eb embedding.Embedder, err error) {
	apiKey, err := g.Cfg().Get(ctx, "doubao_embedding_model.api_key")
	if err != nil {
		return nil, err
	}
	config := &dashscope.EmbeddingConfig{
		APIKey: apiKey.String(),
		Model:  "text-embedding-v4",
	}
	eb, err = dashscope.NewEmbedder(ctx, config)
	if err != nil {
		return nil, err
	}
	return eb, nil
}
