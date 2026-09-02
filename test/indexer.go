package test

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/indexer/redis"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
	redisCli "github.com/redis/go-redis/v9"
)

// newIndexer component initialization function of node 'Indexer' in graph 'KnowledgeIndexing'
func newIndexer(ctx context.Context) (idr indexer.Indexer, err error) {
	addr, err := g.Cfg().Get(ctx, "redis.addr")
	if err != nil {
		return nil, err
	}
	password, err := g.Cfg().Get(ctx, "redis.password")
	if err != nil {
		return nil, err
	}
	client := redisCli.NewClient(&redisCli.Options{
		Addr:     addr.String(),
		Password: password.String(),
	})
	embeddingIns, err := newEmbedding(ctx)
	if err != nil {
		return nil, err
	}
	// 修改配置,保证每个chunk之间不会因为key相同而被覆盖
	var chunkIdx int
	config := &redis.IndexerConfig{
		Client:    client,
		KeyPrefix: "eino_doc:",
		Embedding: embeddingIns,
		DocumentToHashes: func(ctx context.Context, doc *schema.Document) (*redis.Hashes, error) {
			chunkIdx++
			field2Value := map[string]redis.FieldValue{
				"content": {Value: doc.Content, EmbedKey: "vector_content"},
			}
			for k, v := range doc.MetaData {
				field2Value[k] = redis.FieldValue{Value: v}
			}
			return &redis.Hashes{
				Key:         fmt.Sprintf(":%s:%d", doc.ID, chunkIdx),
				Field2Value: field2Value,
			}, nil
		},
	}
	idr, err = redis.NewIndexer(ctx, config)
	if err != nil {
		return nil, err
	}
	return idr, nil
}
