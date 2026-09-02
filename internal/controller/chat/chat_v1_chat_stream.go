package chat

import (
	"OncallAgent/api/chat/v1"
	"OncallAgent/internal/ai/agent/chat_pipeline"
	"OncallAgent/utility/log_call_back"
	"OncallAgent/utility/mem"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
)

func (c *ControllerV1) ChatStream(ctx context.Context, req *v1.ChatStreamReq) (res *v1.ChatStreamRes, err error) {
	id := req.Id
	msg := req.Question

	ctx = context.WithValue(ctx, "client_id", req.Id)
	client, err := c.service.Create(ctx, g.RequestFromCtx(ctx))
	if err != nil {
		return nil, err
	}

	userMessage := &chat_pipeline.UserMessage{
		ID:      id,
		Query:   msg,
		History: mem.GetSimpleMemory(id).GetMessages(),
	}

	runner, err := chat_pipeline.BuildChatAgent(ctx)
	sr, err := runner.Stream(ctx, userMessage, compose.WithCallbacks(log_call_back.LogCallback(nil)))
	if err != nil {
		client.SendToClient("error", err.Error())
		return nil, err
	}
	defer sr.Close()

	var fullResponse strings.Builder

	defer func() {
		completeResponse := fullResponse.String()
		if completeResponse != "" {
			mem.GetSimpleMemory(id).SetMessages(schema.UserMessage(msg))
			mem.GetSimpleMemory(id).SetMessages(schema.SystemMessage(completeResponse))
		}
	}()

	for {
		chunk, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			client.SendToClient("done", "Stream completed")
			g.Log().Infof(ctx, "[chat-stream] full response: %s", fullResponse.String())
			return &v1.ChatStreamRes{}, nil
		}
		if err != nil {
			client.SendToClient("error", err.Error())
			g.Log().Errorf(ctx, "[chat-stream] recv error: %v", err)
			return &v1.ChatStreamRes{}, nil
		}
		g.Log().Debugf(ctx, "[chat-stream] chunk: role=%s tool_calls=%d content=%s",
			chunk.Role, len(chunk.ToolCalls), chunk.Content)
		for _, tc := range chunk.ToolCalls {
			g.Log().Infof(ctx, "[chat-stream] tool_call: name=%s args=%s", tc.Function.Name, tc.Function.Arguments)
		}
		fullResponse.WriteString(chunk.Content)
		client.SendToClient("message", chunk.Content)
	}
}
