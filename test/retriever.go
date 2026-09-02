package test

import (
	"context"
	"strconv"

	"github.com/cloudwego/eino-ext/components/retriever/redis"
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
	redisCli "github.com/redis/go-redis/v9"
)

const (
	//idxname和向量字段必须和index中保持一样,才能被FT读取检索
	// IndexName = "eino_test_idx"
	// VectorField = "vector_content"
	//文档被分为三个chunk?
	TopK = 3
)

func NewRedisRetriever(ctx context.Context) (rtr *redis.Retriever, err error) {
	addr, err := g.Cfg().Get(ctx, "redis.addr")
	if err != nil {
		return nil, err
	}
	password, err := g.Cfg().Get(ctx, "redis.password")
	if err != nil {
		return nil, err
	}
	//使用redis 作为client
	cli := redisCli.NewClient(&redisCli.Options{
		Addr:          addr.String(),
		Password:      password.String(),
		Protocol:      2,
		UnstableResp3: true,
	})
	//和index使用同一个向量模型
	eb, err := newEmbedding(ctx)
	if err != nil {
		return nil, err
	}

	//使用redis的配置实现retriver
	r, err := redis.NewRetriever(ctx, &redis.RetrieverConfig{
		Client:      cli,
		Index:       IndexName,
		VectorField: VectorField,
		TopK:        TopK,
		Embedding:   eb,
		DocumentConverter: func(ctx context.Context, doc redisCli.Document) (*schema.Document, error) {
			resp := &schema.Document{
				ID:      doc.ID,
				Content: doc.Fields["content"],
			}
			dist, _ := strconv.ParseFloat(doc.Fields["distance"], 64)
			resp.WithScore(dist)
			return resp, nil
		},
		ReturnFields: []string{"content", "distance"},
	})
	if err != nil {
		return nil, err
	}
	return r, nil

}
