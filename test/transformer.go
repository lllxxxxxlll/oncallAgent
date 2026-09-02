package test

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino/components/document"
)

// newDocumentTransformer component initialization function of node 'MarkdownSpilter' in graph 'KnowledgeIndexing'
func newDocumentTransformer(ctx context.Context) (tfr document.Transformer, err error) {
	// TODO Modify component configuration here.
	config := &markdown.HeaderConfig{
		Headers: map[string]string{
			"##": "H2",
		},
	}
	tfr, err = markdown.NewHeaderSplitter(ctx, config)
	if err != nil {
		return nil, err
	}
	return tfr, nil
}
