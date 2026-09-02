package log_call_back

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
)

type LogCallbackConfig struct {
	Detail bool
	Debug  bool
}

func LogCallback(config *LogCallbackConfig) callbacks.Handler {
	if config == nil {
		config = &LogCallbackConfig{
			Detail: true,
		}
	}

	builder := callbacks.NewHandlerBuilder()
	builder.OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
		fmt.Printf("[start]:[%s:%s:%s]\n", info.Component, info.Type, info.Name)
		return ctx
	})
	builder.OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
		prefix := fmt.Sprintf("[end]:[%s:%s:%s]", info.Component, info.Type, info.Name)
		if !config.Detail || output == nil {
			fmt.Printf("%s\n", prefix)
			return ctx
		}
		// ChatModel: 只打最后一条 assistant message（包含 tool_calls 或 content）
		if msg, ok := output.(*schema.Message); ok {
			fmt.Printf("%s role=%s content=%q tool_calls=%v\n",
				prefix, msg.Role, truncate(msg.Content, 200), len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				fmt.Printf("  tool_call[%d]: name=%s args=%s\n", i, tc.Function.Name, truncate(tc.Function.Arguments, 300))
			}
			return ctx
		}
		// StreamReader: 流式输出，跳过
		if _, ok := output.(*schema.StreamReader[*schema.Message]); ok {
			fmt.Printf("%s (stream reader, skip detail)\n", prefix)
			return ctx
		}
		b, _ := json.Marshal(output)
		fmt.Printf("%s output=%s\n", prefix, truncate(string(b), 500))
		return ctx
	})
	builder.OnStartWithStreamInputFn(func(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
		fmt.Printf("[stream-start]:[%s:%s:%s]\n", info.Component, info.Type, info.Name)
		return ctx
	})
	builder.OnEndWithStreamOutputFn(func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
		fmt.Printf("[stream-end]:[%s:%s:%s]\n", info.Component, info.Type, info.Name)
		return ctx
	})
	builder.OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
		fmt.Printf("[error]:[%s:%s:%s] err=%v\n", info.Component, info.Type, info.Name, err)
		return ctx
	})
	return builder.Build()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
