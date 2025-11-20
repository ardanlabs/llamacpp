package kronk

import (
	"github.com/hybridgroup/yzma/pkg/llama"
)

// Config represents model level configuration.
type Config struct {
	ContextWindow uint32
	Embeddings    bool
}

func (cfg Config) ctxParams() llama.ContextParams {
	ctxParams := llama.ContextDefaultParams()

	if cfg.Embeddings {
		ctxParams.Embeddings = 1
	}

	if cfg.ContextWindow > 0 {
		ctxParams.NBatch = cfg.ContextWindow
		ctxParams.NUbatch = cfg.ContextWindow
		ctxParams.NCtx = cfg.ContextWindow
	}

	return ctxParams
}
