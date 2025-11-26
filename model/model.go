// Package model provides the low-level api for working with models.
package model

import (
	"context"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hybridgroup/yzma/pkg/llama"
)

type Model struct {
	modelName string
	cfg       Config
	model     llama.Model
	vocab     llama.Vocab
	ctxParams llama.ContextParams
	template  string
	projFile  string
}

func NewModel(modelFile string, projFile string, cfg Config) (*Model, error) {
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

	m := Model{
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

func (m *Model) Unload() {
	llama.ModelFree(m.model)
	llama.BackendFree()
}

func (m *Model) Config() Config {
	return m.cfg
}

func (m *Model) ModelInfo() ModelInfo {
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

func (m *Model) processTokens(ctx context.Context, id string, lctx llama.Context, object string, prompt string, params Params, ch chan<- ChatResponse) {
	var inputTokens int
	var completionTokens int

	params = adjustParams(params)
	sampler, batch, tokenCount := m.startProcessing(lctx, object, prompt, params)

	switch object {
	case ObjectChat:
		inputTokens = tokenCount
	case ObjectVision:
		completionTokens = tokenCount
	}

	totalOutTokens := completionTokens

	// -------------------------------------------------------------------------

	const bufferSize = 32 * 1024
	buf := make([]byte, bufferSize)

	var index int
	var reasoning int
	var completion int
	var reasonTokens int
	var tokensPerSecond float64

	var finalContent strings.Builder
	var finalReasoning strings.Builder

	now := time.Now()

	// -------------------------------------------------------------------------

loop:
	for totalOutTokens <= params.MaxTokens {
		index++

		content, token, err := m.batchResponse(lctx, batch, sampler, buf)
		if err != nil {
			break loop
		}

		// ---------------------------------------------------------------------

		switch content {
		case "<think>":
			batch = m.thinkStart(token, &reasoning, &reasonTokens)
			continue

		case "</think>":
			batch = m.thinkStop(token, &reasoning, &completionTokens)
			continue

		case "<|channel|>":
			batch, content, err = m.channelStart(lctx, token, sampler, buf, &reasoning, &reasonTokens, &completionTokens)
			if err != nil {
				break loop
			}

			if content == "<|continue|>" {
				continue
			}

		case "<|end|>":
			batch, err = m.channelEnd(lctx, token, sampler, buf)
			if err != nil {
				break loop
			}
			continue
		}

		if found := m.removeExtraCRLF(reasoning, completion, content); found {
			batch = m.nextBatch(token)
			continue
		}

		// ---------------------------------------------------------------------

		elapsedSeconds := time.Since(now).Seconds()
		tokensPerSecond = float64(totalOutTokens) / elapsedSeconds

		select {
		case <-ctx.Done():
			ch <- ChatResponseErr(id, object, m.modelName, index, ctx.Err(), Usage{
				InputTokens:      inputTokens,
				ReasoningTokens:  reasonTokens,
				CompletionTokens: completionTokens,
				OutputTokens:     totalOutTokens,
				TokensPerSecond:  tokensPerSecond})
			return

		case ch <- chatResponseDelta(id, object, m.modelName, index, content, reasoning > 0, Usage{
			InputTokens:      inputTokens,
			ReasoningTokens:  reasonTokens,
			CompletionTokens: completionTokens,
			OutputTokens:     totalOutTokens,
			TokensPerSecond:  tokensPerSecond}):
		}

		// ---------------------------------------------------------------------

		batch = m.nextBatch(token)

		switch {
		case reasoning > 0:
			finalReasoning.WriteString(content)
			reasonTokens += int(batch.NTokens)
			reasoning++

		default:
			finalContent.WriteString(content)
			completionTokens += int(batch.NTokens)
			completion++
		}

		totalOutTokens = reasonTokens + completionTokens
	}

	// -------------------------------------------------------------------------

	ch <- chatResponseFinal(id, object, m.modelName, index, finalContent.String(), finalReasoning.String(), Usage{
		InputTokens:      inputTokens,
		ReasoningTokens:  reasonTokens,
		CompletionTokens: completionTokens,
		OutputTokens:     reasonTokens + completionTokens,
		TokensPerSecond:  tokensPerSecond})
}

func (m *Model) startProcessing(lctx llama.Context, object string, prompt string, params Params) (llama.Sampler, llama.Batch, int) {
	sampler := toSampler(params)

	tokens := llama.Tokenize(m.vocab, prompt, true, true)
	batch := llama.BatchGetOne(tokens)
	tokenCount := int(batch.NTokens)

	switch object {
	case ObjectVision:
		batch = m.nextBatch(llama.SamplerSample(sampler, lctx, -1))
		tokenCount = int(batch.NTokens)
	}

	return sampler, batch, tokenCount
}

func (m *Model) nextBatch(token llama.Token) llama.Batch {
	tokens := []llama.Token{token}
	return llama.BatchGetOne(tokens)
}

func (m *Model) batchResponse(lctx llama.Context, batch llama.Batch, sampler llama.Sampler, buf []byte) (string, llama.Token, error) {
	llama.Decode(lctx, batch)
	token := llama.SamplerSample(sampler, lctx, -1)

	if llama.VocabIsEOG(m.vocab, token) {
		return "", 0, io.EOF
	}

	l := llama.TokenToPiece(m.vocab, token, buf, 0, false)

	content := string(buf[:l])
	if content == "" {
		return "", 0, io.EOF
	}

	return content, token, nil
}

func (m *Model) thinkStart(token llama.Token, reasoning *int, reasonTokens *int) llama.Batch {
	*reasoning = 1

	batch := m.nextBatch(token)
	*reasonTokens += int(batch.NTokens)

	return batch
}

func (m *Model) thinkStop(token llama.Token, reasoning *int, completionTokens *int) llama.Batch {
	*reasoning = 0

	batch := m.nextBatch(token)
	*completionTokens += int(batch.NTokens)

	return batch
}

func (m *Model) channelStart(lctx llama.Context, token llama.Token, sampler llama.Sampler, buf []byte, reasoning *int, reasonTokens *int, completionTokens *int) (llama.Batch, string, error) {
	// <|channel|>analysis<|message|>REASONING<|end|><|start|>assistant<|channel|>final<|message|>RESPONSE

	batch := m.nextBatch(token)
	content, token, err := m.batchResponse(lctx, batch, sampler, buf)
	if err != nil {
		return batch, "", err
	}

	switch content {
	case "analysis":
		batch, err = m.channelAnalysis(lctx, token, sampler, buf, reasoning, reasonTokens)
		if err != nil {
			return batch, "", err
		}

	case "final":
		batch, err = m.channelFinal(lctx, token, sampler, buf, reasoning, completionTokens)
		if err != nil {
			return batch, "", err
		}

	default:
		return batch, content, nil
	}

	return batch, "<|continue|>", nil
}

func (m *Model) channelAnalysis(lctx llama.Context, token llama.Token, sampler llama.Sampler, buf []byte, reasoning *int, reasonTokens *int) (llama.Batch, error) {
	*reasoning = 1

	batch := m.nextBatch(token)
	_, token, err := m.batchResponse(lctx, batch, sampler, buf) // <|message|>
	if err != nil {
		return batch, err
	}

	batch = m.nextBatch(token)
	*reasonTokens += int(batch.NTokens)

	return batch, nil
}

func (m *Model) channelFinal(lctx llama.Context, token llama.Token, sampler llama.Sampler, buf []byte, reasoning *int, completionTokens *int) (llama.Batch, error) {
	*reasoning = 0

	batch := m.nextBatch(token)
	_, token, err := m.batchResponse(lctx, batch, sampler, buf) // <|message|>
	if err != nil {
		return batch, err
	}

	batch = m.nextBatch(token)
	*completionTokens += int(batch.NTokens)

	return batch, nil
}

func (m *Model) channelEnd(lctx llama.Context, token llama.Token, sampler llama.Sampler, buf []byte) (llama.Batch, error) {
	batch := m.nextBatch(token)

	_, token, err := m.batchResponse(lctx, batch, sampler, buf) // <|start|>
	if err != nil {
		return batch, err
	}

	batch = m.nextBatch(token)

	_, token, err = m.batchResponse(lctx, batch, sampler, buf) // assistant
	if err != nil {
		return batch, err
	}

	batch = m.nextBatch(token)
	return batch, nil
}

func (m *Model) removeExtraCRLF(reasoning int, completion int, content string) bool {
	// We just started reasoning so remove leading CR.
	if reasoning == 1 && content == "\x0A" {
		return true
	}

	// We just started completion so remove leading CR.
	if reasoning == 0 && completion == 0 && (content == "\x0A\x0A" || content == "\x0A") {
		return true
	}

	return false
}
