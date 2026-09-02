package main

import (
	"context"
	"log"

	"OncallAgent/test"
)

// 一次性的索引 provisioning 入口:只负责建索引,不负责写数据。
// 对比 Milvus:那边 collection/schema/index 是 SDK 托管的一个受管对象;
// 这边你得单独跑这一步,把 FT.CREATE 和 HSET 两个割裂的动作手动串起来。
func main() {
	ctx := context.Background()
	if err := test.CreateIndex(ctx); err != nil {
		log.Fatalf("CreateIndex failed: %v", err)
	}
	log.Println("✅ 索引已建立。验证: redis-cli -p 6370 FT.INFO eino_test_idx | grep num_docs")
}
