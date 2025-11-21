package kronk

import (
	"context"
	"fmt"

	"github.com/hybridgroup/yzma/pkg/llama"
)

func (m *model) chatStreaming(ctx context.Context, messages []ChatMessage, params Params) <-chan ChatResponse {
	ch := make(chan ChatResponse)

	go func() {
		defer close(ch)

		lctx, err := llama.InitFromModel(m.model, m.ctxParams)
		if err != nil {
			ch <- ChatResponse{Err: fmt.Errorf("unable to init from model: %w", err)}
			return
		}

		defer func() {
			llama.Synchronize(lctx)
			llama.Free(lctx)
		}()

		prompt := m.applyChatTemplate(messages)
		m.processChatStreaming(ctx, lctx, prompt, toSampler(params), ch)
	}()

	return ch
}

func (m *model) applyChatTemplate(messages []ChatMessage) string {
	msgs := make([]llama.ChatMessage, len(messages))
	for i, msg := range messages {
		msgs[i] = llama.NewChatMessage(msg.Role, msg.Content)
	}

	buf := make([]byte, 1024*32)
	l := llama.ChatApplyTemplate(m.template, msgs, true, buf)

	return string(buf[:l])
}

func (m *model) processChatStreaming(ctx context.Context, lctx llama.Context, prompt string, sampler llama.Sampler, ch chan<- ChatResponse) {
	tokens := llama.Tokenize(m.vocab, prompt, true, true)
	buf := make([]byte, 1024*32)

	for range llama.MaxToken {
		select {
		case <-ctx.Done():
			ch <- ChatResponse{Err: ctx.Err()}
			return
		default:
		}

		batch := llama.BatchGetOne(tokens)
		llama.Decode(lctx, batch)

		token := llama.SamplerSample(sampler, lctx, -1)

		if llama.VocabIsEOG(m.vocab, token) {
			break
		}

		l := llama.TokenToPiece(m.vocab, token, buf, 0, false)

		resp := string(buf[:l])
		if resp == "" {
			break
		}

		select {
		case <-ctx.Done():
			ch <- ChatResponse{Err: ctx.Err()}
			return

		case ch <- ChatResponse{Response: resp}:
		}

		tokens = []llama.Token{token}
	}
}
