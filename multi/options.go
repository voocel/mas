package multi

import (
	"sync"

	"github.com/voocel/mas/memory"
)

type Option func(*runConfig)

type runConfig struct {
	sharedMemory        memory.Store
	sharedMu            *sync.Mutex
	maxSteps            int
	enableTransferTools bool
}

type HandoffOption = Option

func defaultRunConfig() runConfig {
	return runConfig{maxSteps: 3, enableTransferTools: true}
}

func applyOptions(opts ...Option) runConfig {
	cfg := defaultRunConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.sharedMemory != nil && cfg.sharedMu == nil {
		cfg.sharedMu = &sync.Mutex{}
	}
	return cfg
}

func WithSharedMemory(store memory.Store) Option {
	return func(cfg *runConfig) {
		cfg.sharedMemory = store
	}
}

func WithMaxSteps(steps int) Option {
	return func(cfg *runConfig) {
		if steps > 0 {
			cfg.maxSteps = steps
		}
	}
}

func WithTransferTools(enabled bool) Option {
	return func(cfg *runConfig) {
		cfg.enableTransferTools = enabled
	}
}
