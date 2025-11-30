package model

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hybridgroup/yzma/pkg/llama"
)

// Chat performs a chat request and returns the final response.
func (m *Model) Chat(ctx context.Context, params Params, d D) (ChatResponse, error) {
	ch := m.ChatStreaming(ctx, params, d)

	var lastMsg ChatResponse
	for msg := range ch {
		lastMsg = msg
	}

	return lastMsg, nil
}

// ChatStreaming performs a chat request and streams the response.
func (m *Model) ChatStreaming(ctx context.Context, params Params, d D) <-chan ChatResponse {
	ch := make(chan ChatResponse)

	go func() {
		m.activeStreams.Add(1)
		defer m.activeStreams.Add(-1)

		id := uuid.New().String()

		defer func() {
			if rec := recover(); rec != nil {
				m.sendChatError(ctx, ch, id, fmt.Errorf("%v", rec))
			}
			close(ch)
		}()

		lctx, err := llama.InitFromModel(m.model, m.ctxParams)
		if err != nil {
			m.sendChatError(ctx, ch, id, fmt.Errorf("unable to init from model: %w", err))
			return
		}

		defer func() {
			llama.Synchronize(lctx)
			llama.Free(lctx)
		}()

		prompt, err := m.applyJinjaTemplate(d)
		if err != nil {
			m.sendChatError(ctx, ch, id, fmt.Errorf("unable to apply jinja template: %w", err))
			return
		}

		m.processTokens(ctx, id, lctx, ObjectChat, prompt, params, ch)
	}()

	return ch
}

func (m *Model) sendChatError(ctx context.Context, ch chan<- ChatResponse, id string, err error) {
	// I want to try and send this message before we check the context.
	select {
	case ch <- ChatResponseErr(id, ObjectChat, m.modelInfo.Name, 0, err, Usage{}):
		return
	default:
	}

	select {
	case <-ctx.Done():
		select {
		case ch <- ChatResponseErr(id, ObjectChat, m.modelInfo.Name, 0, ctx.Err(), Usage{}):
		default:
		}

	case ch <- ChatResponseErr(id, ObjectChat, m.modelInfo.Name, 0, err, Usage{}):
	}
}
