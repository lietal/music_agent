package api

import "context"

type ctxKey int

const (
	ctxKeyRunID ctxKey = iota
	ctxKeyConvID
)

func withRunID(ctx context.Context, runID string) context.Context {
	return context.WithValue(ctx, ctxKeyRunID, runID)
}

func runIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyRunID).(string); ok {
		return v
	}
	return ""
}

func withConvID(ctx context.Context, convID string) context.Context {
	return context.WithValue(ctx, ctxKeyConvID, convID)
}

func convIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyConvID).(string); ok {
		return v
	}
	return ""
}
