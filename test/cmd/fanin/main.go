package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino/compose"
)

func BuildFanInPipeline(ctx context.Context) (compose.Runnable[string, string], error) {
	g := compose.NewGraph[string, string]()

	// 分支 A：生成问候语 + 设 OutputKey="greeting"
	_ = g.AddLambdaNode("GreetingNode",
		compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
			return "你好, " + input, nil
		}),
		compose.WithOutputKey("greeting"), // ← 输出: map["greeting"]="你好, xxx"
	)

	// 分支 B：生成大写版本 + 设 OutputKey="upper"
	_ = g.AddLambdaNode("UpperNode",
		compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
			return strings.ToUpper(input), nil
		}),
		compose.WithOutputKey("upper"), // ← 输出: map["upper"]="WORLD"
	)

	// 汇聚节点：接收合并后的 map，从中提取两个分支的结果
	_ = g.AddLambdaNode("Merger",
		compose.InvokableLambda(func(ctx context.Context, input map[string]any) (string, error) {
			// input = {"greeting": "你好, world", "upper": "WORLD"}
			greeting := input["greeting"].(string)
			upper := input["upper"].(string)
			return fmt.Sprintf("%s [%s]", greeting, upper), nil
		}),
		// 不设 Key → 输入类型就是 map[string]any
	)

	// 拓扑: START → GreetingNode ──┐
	//       START → UpperNode ─────┤
	//                               ├→ Merger → END
	_ = g.AddEdge(compose.START, "GreetingNode")
	_ = g.AddEdge(compose.START, "UpperNode")
	_ = g.AddEdge("GreetingNode", "Merger")
	_ = g.AddEdge("UpperNode", "Merger")
	_ = g.AddEdge("Merger", compose.END)

	return g.Compile(ctx, compose.WithGraphName("FanInDemo"))
}

func main() {
	ctx := context.Background()
	runner, err := BuildFanInPipeline(ctx)
	if err != nil {
		log.Fatal(err)
	}

	result, err := runner.Invoke(ctx, "world")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
	// 输出: 你好, world [WORLD]
}
