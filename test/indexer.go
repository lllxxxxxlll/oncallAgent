package test

import (
	"context"

	"github.com/cloudwego/eino-ext/components/indexer/redis"
	"github.com/cloudwego/eino/components/indexer"
	redisCli "github.com/redis/go-redis/v9"
)

// newIndexer component initialization function of node 'Indexer' in graph 'KnowledgeIndexing'
func newIndexer(ctx context.Context) (idr indexer.Indexer, err error) {
	client := redisCli.NewClient(&redisCli.Options{
		Addr:     "localhost:6379",
		Password: "pioneer..,,",
	})
	embeddingIns, err := newEmbedding(ctx)
	if err != nil {
		return nil, err
	}
	config := &redis.IndexerConfig{
		Client:    client,
		KeyPrefix: "eino_doc:",
		Embedding: embeddingIns,
	}
	idr, err = redis.NewIndexer(ctx, config)
	if err != nil {
		return nil, err
	}
	return idr, nil
}
