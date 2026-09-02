package test2

import (
	"context"

	"github.com/cloudwego/eino/compose"
)

func Buildfile(ctx context.Context) (r compose.Runnable[any, any], err error) {
	const (
		Loader1              = "Loader1"
		DocumentTransformer1 = "DocumentTransformer1"
	)
	g := compose.NewGraph[any, any]()
	loader1KeyOfLoader, err := newLoader(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddLoaderNode(Loader1, loader1KeyOfLoader, compose.WithOutputKey("docs"))
	documentTransformer1KeyOfDocumentTransformer, err := newDocumentTransformer(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddDocumentTransformerNode(DocumentTransformer1, documentTransformer1KeyOfDocumentTransformer, compose.WithInputKey("docs"), compose.WithOutputKey("docs"))
	_ = g.AddEdge(compose.START, Loader1)
	_ = g.AddEdge(DocumentTransformer1, compose.END)
	_ = g.AddEdge(Loader1, DocumentTransformer1)
	r, err = g.Compile(ctx, compose.WithGraphName("file"), compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		return nil, err
	}
	return r, err
}
