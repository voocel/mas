package multi

import (
	"context"

	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/schema"
)

func prepareRun(ctx context.Context, r *runner.Runner, cfg runConfig) (*runner.Runner, error) {
	run := r
	if r != nil && r.GetMemory() != nil {
		run = r.WithMemory(r.GetMemory().Clone())
	}
	if cfg.sharedMemory == nil {
		return run, nil
	}
	history, err := cfg.sharedMemory.History(ctx)
	if err != nil {
		return nil, err
	}
	if len(history) == 0 {
		return run, nil
	}
	if run.GetMemory() == nil {
		return run, nil
	}
	if err := run.GetMemory().AddBatch(ctx, history); err != nil {
		return nil, err
	}
	return run, nil
}

func appendShared(ctx context.Context, cfg runConfig, message schema.Message) error {
	if cfg.sharedMemory == nil {
		return nil
	}
	if message.Role == "" {
		return nil
	}
	if cfg.sharedMu != nil {
		cfg.sharedMu.Lock()
		defer cfg.sharedMu.Unlock()
	}
	return cfg.sharedMemory.Add(ctx, message)
}
