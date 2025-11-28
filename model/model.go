// Package model provides the low-level api for working with models.
package model

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hybridgroup/yzma/pkg/llama"
)

// Model represents a model and provides a low-level API for working with it.
type Model struct {
	cfg       Config
	model     llama.Model
	vocab     llama.Vocab
	ctxParams llama.ContextParams
	template  string
	projFile  string
	modelInfo ModelInfo
}

func NewModel(cfg Config) (*Model, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("unable to validate config: %w", err)
	}

	mparams := llama.ModelDefaultParams()
	if cfg.Device != "" {
		dev := llama.GGMLBackendDeviceByName(cfg.Device)
		if dev == 0 {
			return nil, fmt.Errorf("unknown device: %s", cfg.Device)
		}
		mparams.SetDevices([]llama.GGMLBackendDevice{dev})
	}

	mdl, err := llama.ModelLoadFromFile(cfg.ModelFile, mparams)
	if err != nil {
		return nil, fmt.Errorf("unable to load model: %w", err)
	}

	cfg = adjustConfig(cfg, mdl)
	vocab := llama.ModelGetVocab(mdl)

	// -------------------------------------------------------------------------

	template, err := retrieveTemplate(cfg, mdl)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve model template: %w", err)
	}

	// -------------------------------------------------------------------------

	m := Model{
		cfg:       cfg,
		model:     mdl,
		vocab:     vocab,
		ctxParams: modelCtxParams(cfg),
		template:  template,
		projFile:  cfg.ProjectionFile,
		modelInfo: newModelInfo(cfg, mdl),
	}

	return &m, nil
}

func retrieveTemplate(cfg Config, mdl llama.Model) (string, error) {
	var template string

	if cfg.JinjaFile != "" {
		template, err := readJinjaTemplate(cfg.JinjaFile)
		if err != nil {
			return "", fmt.Errorf("failed to read jinja template: %w", err)
		}

		if template == "" {
			return "", fmt.Errorf("jinja template is empty")
		}
	}

	if template == "" {
		template = llama.ModelChatTemplate(mdl, "")
		if template == "" {
			template, _ = llama.ModelMetaValStr(mdl, "tokenizer.chat_template")
		}
	}

	return template, nil
}

func (m *Model) Unload() {
	llama.ModelFree(m.model)
	llama.BackendFree()
}

func (m *Model) Config() Config {
	return m.cfg
}

func (m *Model) ModelInfo() ModelInfo {
	return m.modelInfo
}

func (m *Model) processTokens(ctx context.Context, id string, lctx llama.Context, object string, prompt string, params Params, ch chan<- ChatResponse) {
	var inputTokens int
	var completionTokens int

	params = adjustParams(params)
	sampler, batch, inputTokens, completionTokens := m.startProcessing(lctx, object, prompt, params)

	totalOutTokens := completionTokens

	// -------------------------------------------------------------------------

	const bufferSize = 32 * 1024
	buf := make([]byte, bufferSize)

	var index int
	var reasoning int
	var completion int
	var tooling int
	var reasonTokens int
	var tokensPerSecond float64

	var finalReasoning strings.Builder
	var finalContent strings.Builder
	var finalTooling strings.Builder

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

		case "<tool_call>":
			batch = m.toolCallStart(token, &tooling, &completionTokens)
			continue

		case "</tool_call>":
			batch = m.toolCallStop(token, &tooling, &completionTokens)
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

		if found := m.removeExtraCRLF(reasoning, tooling, completion, content); found {
			batch = m.nextBatch(token)
			continue
		}

		// ---------------------------------------------------------------------

		elapsedSeconds := time.Since(now).Seconds()
		tokensPerSecond = float64(totalOutTokens) / elapsedSeconds

		// We want to return the tool calling in a single response to make
		// it easier for developers to process. We will collect all of
		// this and this should be the final response.
		if tooling > 0 {
			finalTooling.WriteString(content)
			completionTokens += int(batch.NTokens)
			totalOutTokens = reasonTokens + completionTokens

			batch = m.nextBatch(token)
			continue
		}

		select {
		case <-ctx.Done():
			ch <- ChatResponseErr(id, object, m.modelInfo.Name, index, ctx.Err(), Usage{
				InputTokens:      inputTokens,
				ReasoningTokens:  reasonTokens,
				CompletionTokens: completionTokens,
				OutputTokens:     totalOutTokens,
				TokensPerSecond:  tokensPerSecond})
			return

		case ch <- chatResponseDelta(id, object, m.modelInfo.Name, index, content, reasoning > 0, Usage{
			InputTokens:      inputTokens,
			ReasoningTokens:  reasonTokens,
			CompletionTokens: completionTokens,
			OutputTokens:     totalOutTokens,
			TokensPerSecond:  tokensPerSecond}):
		}

		// ---------------------------------------------------------------------

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

		batch = m.nextBatch(token)
	}

	// -------------------------------------------------------------------------

	// We will add an ID to this tool call to help the model when the
	// user returns the tool call response.
	var toolingContent string
	if finalTooling.Len() > 0 {
		toolingContent = addToolingID(finalTooling)
	}

	ch <- chatResponseFinal(
		id,
		object,
		m.modelInfo.Name,
		index,
		finalContent.String(),
		finalReasoning.String(),
		toolingContent,
		Usage{
			InputTokens:      inputTokens,
			ReasoningTokens:  reasonTokens,
			CompletionTokens: completionTokens,
			OutputTokens:     reasonTokens + completionTokens,
			TokensPerSecond:  tokensPerSecond},
	)
}

func (m *Model) startProcessing(lctx llama.Context, object string, prompt string, params Params) (llama.Sampler, llama.Batch, int, int) {
	sampler := toSampler(params)

	tokens := llama.Tokenize(m.vocab, prompt, true, true)
	batch := llama.BatchGetOne(tokens)
	inpTokenCount := int(batch.NTokens)

	var outTokenCount int
	switch object {
	case ObjectVision:
		batch = m.nextBatch(llama.SamplerSample(sampler, lctx, -1))
		outTokenCount = int(batch.NTokens)
	}

	return sampler, batch, inpTokenCount, outTokenCount
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

func (m *Model) toolCallStart(token llama.Token, tooling *int, completionTokens *int) llama.Batch {
	*tooling = 1

	batch := m.nextBatch(token)
	*completionTokens += int(batch.NTokens)

	return batch
}

func (m *Model) toolCallStop(token llama.Token, tooling *int, completionTokens *int) llama.Batch {
	*tooling = 0

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

func (m *Model) removeExtraCRLF(reasoning int, tooling int, completion int, content string) bool {
	// We just started reasoning or tool calling so remove leading CR.
	if (reasoning == 1 || tooling == 1) && content == "\x0A" {
		return true
	}

	// We just started completion so remove leading CR.
	if reasoning == 0 && tooling == 0 && completion == 0 && (content == "\x0A\x0A" || content == "\x0A") {
		return true
	}

	return false
}

// =============================================================================

func addToolingID(tooling strings.Builder) string {
	var toolCall struct {
		ID        string         `json:"id"`
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}

	str := tooling.String()

	if err := json.Unmarshal([]byte(str), &toolCall); err != nil {
		return str
	}

	toolCall.ID = uuid.NewString()

	data, err := json.Marshal(toolCall)
	if err != nil {
		return str
	}

	return string(data)
}
