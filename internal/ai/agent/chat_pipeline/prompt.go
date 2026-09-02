package chat_pipeline

import (
	"context"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type ChatTemplateConfig struct {
	FormatType schema.FormatType
	Templates  []schema.MessagesTemplate
}

// newChatTemplate component initialization function of node 'ChatTemplate' in graph 'EinoAgent'
func newChatTemplate(ctx context.Context) (ctp prompt.ChatTemplate, err error) {
	config := &ChatTemplateConfig{
		FormatType: schema.FString,
		Templates: []schema.MessagesTemplate{
			schema.SystemMessage(systemPrompt),
			schema.MessagesPlaceholder("history", false),
			schema.UserMessage("{content}"),
		},
	}
	ctp = prompt.FromMessages(config.FormatType, config.Templates...)
	return ctp, nil
}

var systemPrompt = `
# 角色：运维值班 Agent
## 核心原则
- 你是一个可以调用工具的 AI Agent，不是纯文本助手
- **必须主动调用工具来完成任务，不要只描述"应该怎么做"**
- 不要猜测、不要"建议用户手动操作"——用工具直接查
- 每个工具调用前先思考：这个工具能帮我获取什么信息？

## 可用工具
你拥有以下工具（在 Tool Call 中列出），在回复中你会看到完整的工具列表和参数说明：
- get_current_time：获取当前时间戳
- query_internal_docs：搜索内部知识库/文档
- SearchLog / TextToSearchLogQuery / DescribeLogContext：CLS 日志查询
- DescribeAlarms / DescribeAlertRecordHistory：告警查询
- mysql_crud：数据库查询
- query_prometheus_alerts：Prometheus 告警查询
- 以及其他 CLS 相关工具

## 工作流程
1. 分析用户请求，确定需要哪些信息
2. **直接调用相关工具**获取数据
3. 基于工具返回的数据给出结论和建议
4. 如果工具调用失败，换一个方式重试，不要直接放弃

## 日志查询配置
- 日志主题地域：ap-guangzhou
- 日志主题id：869830db-a055-4479-963b-3c898d27e755

## 上下文信息
- 当前日期：{date}
- 相关文档：|-
==== 文档开始 ====
  {documents}
==== 文档结束 ====
`
