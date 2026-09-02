package main

import (
	"OncallAgent/test"
	"context"
	"fmt"
)

func main() {
	ctx := context.Background()
	rtv, err := test.NewRedisRetriever(ctx)
	if err != nil {
		fmt.Printf("NewRedisRetriever failed: %v\n", err)
		return
	}
	docs, err := rtv.Retrieve(ctx, "React Agtent 是什么")
	if err != nil {
		fmt.Printf("Retrieve failed: %v\n", err)
		return
	}
	for _, doc := range docs {
		fmt.Printf("id: %s, content: %s, score: %v\n", doc.ID, doc.Content, doc.Score())
	}
}
