package observer

import (
	"context"

	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runner"
)

// CompositeObserver combines multiple observers.
type CompositeObserver struct {
	items []runner.Observer
}

// NewCompositeObserver creates a composite observer.
func NewCompositeObserver(items ...runner.Observer) *CompositeObserver {
	return &CompositeObserver{items: filterObservers(items)}
}

// Add appends observers.
func (o *CompositeObserver) Add(items ...runner.Observer) {
	o.items = append(o.items, filterObservers(items)...)
}

func (o *CompositeObserver) OnLLMStart(ctx context.Context, state *runner.State, req *llm.Request) {
	for _, obs := range o.items {
		obs.OnLLMStart(ctx, state, req)
	}
}

func (o *CompositeObserver) OnLLMEnd(ctx context.Context, state *runner.State, resp *llm.Response, err error) {
	for _, obs := range o.items {
		obs.OnLLMEnd(ctx, state, resp, err)
	}
}

func (o *CompositeObserver) OnToolCall(ctx context.Context, state *runner.ToolState) {
	for _, obs := range o.items {
		obs.OnToolCall(ctx, state)
	}
}

func (o *CompositeObserver) OnToolResult(ctx context.Context, state *runner.ToolState) {
	for _, obs := range o.items {
		obs.OnToolResult(ctx, state)
	}
}

func (o *CompositeObserver) OnError(ctx context.Context, err error) {
	for _, obs := range o.items {
		obs.OnError(ctx, err)
	}
}

func filterObservers(items []runner.Observer) []runner.Observer {
	result := make([]runner.Observer, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, item)
		}
	}
	return result
}

var _ runner.Observer = (*CompositeObserver)(nil)
