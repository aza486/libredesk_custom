package main

import (
	"context"

	"github.com/abhinavxd/libredesk/internal/learning"
)

// Implemented by the existing aiassist.Client with its protected transport.
type learningEmbeddingCaller interface {
	Call(context.Context, string, string, any, any) (int, error)
}

// Shared unchanged embedding contract for the fact and separate case publishers.
func newLearningEmbedder(client learningEmbeddingCaller) learning.Embedder {
	return func(ctx context.Context, text string) ([]float64, error) {
		var result struct {
			Data []struct {
				Embedding []float64 `json:"embedding"`
			} `json:"data"`
		}
		status, err := client.Call(ctx, "Authorization", "", map[string]any{"model": "text-embedding-3-small", "dimensions": 1536, "input": text}, &result)
		if err != nil {
			return nil, learningDependencyFailure("embedding", status, err)
		}
		if len(result.Data) != 1 {
			return nil, &learning.DependencyError{Service: "embedding", Kind: "invalid_response"}
		}
		return result.Data[0].Embedding, nil
	}
}
