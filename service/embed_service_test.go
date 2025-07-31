package service

import (
	"context"
	"testing"
)

func TestEntry(t *testing.T) {
	ctx := context.Background()
	repoPath := "/Users/silhouette/codeworks/static_parser"
	TraceAllRepoFunc(ctx, repoPath)
}
