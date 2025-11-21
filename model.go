package kronk

import (
	"fmt"
	"sync"

	"github.com/hybridgroup/yzma/pkg/llama"
)

type model struct {
	model     llama.Model
	vocab     llama.Vocab
	ctxParams llama.ContextParams
	template  string
	projFile  string
	muHEC     sync.Mutex
}

func newModel(modelFile string, cfg Config, options ...func(m *model) error) (*model, error) {
	mdl, err := llama.ModelLoadFromFile(modelFile, llama.ModelDefaultParams())
	if err != nil {
		return nil, fmt.Errorf("ModelLoadFromFile: %w", err)
	}

	vocab := llama.ModelGetVocab(mdl)

	// -------------------------------------------------------------------------

	template := llama.ModelChatTemplate(mdl, "")
	if template == "" {
		template, _ = llama.ModelMetaValStr(mdl, "tokenizer.chat_template")
	}

	if template == "" {
		template = "chatml"
	}

	// -------------------------------------------------------------------------

	m := model{
		model:     mdl,
		vocab:     vocab,
		ctxParams: cfg.ctxParams(),
		template:  template,
	}

	for _, option := range options {
		if err := option(&m); err != nil {
			llama.ModelFree(mdl)
			llama.BackendFree()
			return nil, err
		}
	}

	return &m, nil
}

func (m *model) unload() {
	llama.ModelFree(m.model)
	llama.BackendFree()
}

func (m *model) modelInfo() ModelInfo {
	desc := llama.ModelDesc(m.model)
	size := llama.ModelSize(m.model)
	encoder := llama.ModelHasEncoder(m.model)
	decoder := llama.ModelHasDecoder(m.model)
	recurrent := llama.ModelIsRecurrent(m.model)
	hybrid := llama.ModelIsHybrid(m.model)
	count := llama.ModelMetaCount(m.model)
	metadata := make(map[string]string)

	for i := range count {
		key, ok := llama.ModelMetaKeyByIndex(m.model, i)
		if !ok {
			continue
		}

		value, ok := llama.ModelMetaValStrByIndex(m.model, i)
		if !ok {
			continue
		}

		metadata[key] = value
	}

	return ModelInfo{
		Desc:        desc,
		Size:        size,
		HasEncoder:  encoder,
		HasDecoder:  decoder,
		IsRecurrent: recurrent,
		IsHybrid:    hybrid,
		Metadata:    metadata,
	}
}
