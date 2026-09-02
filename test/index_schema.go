package test

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/frame/g"
	redisCli "github.com/redis/go-redis/v9"
)

// 索引与写入侧共享的常量,三处必须对齐,错一处就静默查不到:
//   - KeyPrefix   == test/indexer.go 里 IndexerConfig.KeyPrefix
//   - VectorField == test/indexer.go 里 DocumentToHashes 的 EmbedKey
//   - VectorDim   == embedding 真实维度(实测 HSTRLEN vector_content = 4096 字节 / 4)
const (
	IndexName   = "eino_test_idx"  // 检索侧 retriever 必须用同一个名字
	KeyPrefix   = "eino_doc:"      // 索引哪些 key
	VectorField = "vector_content" // 向量存在哪个 hash field 下
	VectorDim   = 1024
)

// CreateIndex 建立 RediSearch 向量索引。
//
// 这是 Milvus 帮你托管、而 Redis Stack 逼你手写的那一层:
// eino-ext 的 redis indexer 只做 HSET 写入(indexer.go:126 pipelineHSet),
// 索引结构(FT.CREATE)完全是调用方的责任。普通 Redis 连这条命令都不认识——
// 向量检索是 RediSearch 模块提供的外挂能力,不是 Redis 本体。
func CreateIndex(ctx context.Context) error {
	addr, err := g.Cfg().Get(ctx, "redis.addr")
	if err != nil {
		return err
	}
	password, err := g.Cfg().Get(ctx, "redis.password")
	if err != nil {
		return err
	}
	client := redisCli.NewClient(&redisCli.Options{
		Addr:     addr.String(),
		Password: password.String(),
	})
	defer client.Close()

	// 幂等:先删旧索引。首次不存在会返回 "Unknown Index name",忽略即可。
	// FTDropIndex 不带 DD 参数 → 只拆索引,不动底层那 3 个 hash 数据。
	_ = client.FTDropIndex(ctx, IndexName).Err()

	schemas := []*redisCli.FieldSchema{
		{
			FieldName: "content",
			FieldType: redisCli.SearchFieldTypeText,
		},
		{
			FieldName: VectorField,
			FieldType: redisCli.SearchFieldTypeVector,
			VectorArgs: &redisCli.FTVectorArgs{
				// FLAT:精确暴力检索,适合 < 1M 向量的小数据集。
				// 想体会"近似 vs 精确",把这里换成 HNSWOptions 再对比召回。
				// FlatOptions: &redisCli.FTFlatOptions{
				// 	Type:           "FLOAT32", // 对齐 utils.go vector2Bytes 的 float32 小端序列化
				// 	Dim:            VectorDim,  // 对齐 embedding 维度,错一位查出来全是噪声
				// 	DistanceMetric: "COSINE",   // 对齐检索侧度量:L2 / IP / COSINE
				// },
				HNSWOptions: &redisCli.FTHNSWOptions{
					Type:           "FLOAT32",
					Dim:            VectorDim,
					DistanceMetric: "COSINE",
				},
			},
		},
	}

	// OnHash + Prefix:告诉 RediSearch "去索引所有 eino_doc: 开头的 hash"。
	res, err := client.FTCreate(ctx, IndexName,
		&redisCli.FTCreateOptions{
			OnHash: true,
			Prefix: []any{KeyPrefix},
		},
		schemas...,
	).Result()
	if err != nil {
		return fmt.Errorf("FT.CREATE failed: %w", err)
	}
	fmt.Printf("FT.CREATE %s -> %s\n", IndexName, res)
	return nil
}
