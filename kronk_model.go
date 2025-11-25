package kronk

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hybridgroup/yzma/pkg/llama"
)

type model struct {
	modelName string
	cfg       ModelConfig
	model     llama.Model
	vocab     llama.Vocab
	ctxParams llama.ContextParams
	template  string
	projFile  string
}

func newModel(modelFile string, projFile string, cfg ModelConfig) (*model, error) {
	mparams := llama.ModelDefaultParams()
	if cfg.Device != "" {
		dev := llama.GGMLBackendDeviceByName(cfg.Device)
		if dev == 0 {
			return nil, fmt.Errorf("unknown device: %s", cfg.Device)
		}
		mparams.SetDevices([]llama.GGMLBackendDevice{dev})
	}

	mdl, err := llama.ModelLoadFromFile(modelFile, mparams)
	if err != nil {
		return nil, fmt.Errorf("ModelLoadFromFile: %w", err)
	}

	cfg = adjustConfig(cfg, mdl)

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

	filename := filepath.Base(modelFile)
	modelName := strings.TrimSuffix(filename, path.Ext(filename))

	// -------------------------------------------------------------------------

	m := model{
		modelName: modelName,
		cfg:       cfg,
		model:     mdl,
		vocab:     vocab,
		ctxParams: modelCtxParams(cfg),
		template:  template,
		projFile:  projFile,
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

func (m *model) processTokens(ctx context.Context, id string, lctx llama.Context, object string, prompt string, params Params, ch chan<- ChatResponse) {
	var inputTokens int
	var completionTokens int
	var reasonTokens int

	var tokens []llama.Token
	var batch llama.Batch
	var finalContent strings.Builder

	// -------------------------------------------------------------------------

	params = adjustParams(params)
	sampler := toSampler(params)

	tokens = llama.Tokenize(m.vocab, prompt, true, true)
	batch = llama.BatchGetOne(tokens)
	inputTokens = int(batch.NTokens)

	switch object {
	case ObjectVision:
		tokens = []llama.Token{llama.SamplerSample(sampler, lctx, -1)}
		batch = llama.BatchGetOne(tokens)

		completionTokens = int(batch.NTokens)
	}

	// -------------------------------------------------------------------------

	const bufferSize = 32 * 1024
	buf := make([]byte, bufferSize)

	var index int
	var tokensPerSecond float64
	totalOutTokens := reasonTokens + completionTokens

	now := time.Now()

	// -------------------------------------------------------------------------

	for totalOutTokens <= params.MaxTokens {
		index++

		llama.Decode(lctx, batch)
		token := llama.SamplerSample(sampler, lctx, -1)

		if llama.VocabIsEOG(m.vocab, token) {
			break
		}

		l := llama.TokenToPiece(m.vocab, token, buf, 0, false)

		content := string(buf[:l])
		if content == "" {
			break
		}

		finalContent.WriteString(content)

		elapsedSeconds := time.Since(now).Seconds()
		tokensPerSecond = float64(totalOutTokens) / elapsedSeconds

		select {
		case <-ctx.Done():
			ch <- chatResponseErr(id, object, m.modelName, index, ctx.Err(), Usage{
				InputTokens:      inputTokens,
				ReasoningTokens:  reasonTokens,
				CompletionTokens: completionTokens,
				OutputTokens:     totalOutTokens,
				TokensPerSecond:  tokensPerSecond})
			return

		case ch <- chatResponseDelta(id, object, m.modelName, index, content, Usage{
			InputTokens:      inputTokens,
			ReasoningTokens:  reasonTokens,
			CompletionTokens: completionTokens,
			OutputTokens:     totalOutTokens,
			TokensPerSecond:  tokensPerSecond}):
		}

		tokens = []llama.Token{token}
		batch = llama.BatchGetOne(tokens)

		completionTokens += int(batch.NTokens)
		totalOutTokens = reasonTokens + completionTokens
	}

	// -------------------------------------------------------------------------

	ch <- chatResponseFinal(id, object, m.modelName, index, finalContent.String(), Usage{
		InputTokens:      inputTokens,
		ReasoningTokens:  reasonTokens,
		CompletionTokens: completionTokens,
		OutputTokens:     reasonTokens + completionTokens,
		TokensPerSecond:  tokensPerSecond})
}
