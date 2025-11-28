package model

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hybridgroup/yzma/pkg/llama"
)

// Chat performs a chat request and returns the final response.
func (m *Model) Chat(ctx context.Context, cr ChatRequest) (ChatResponse, error) {
	ch := m.ChatStreaming(ctx, cr)

	var lastMsg ChatResponse
	for msg := range ch {
		lastMsg = msg
	}

	return lastMsg, nil
}

// ChatStreaming performs a chat request and streams the response.
func (m *Model) ChatStreaming(ctx context.Context, cr ChatRequest) <-chan ChatResponse {
	ch := make(chan ChatResponse)

	go func() {
		id := uuid.New().String()

		defer func() {
			if rec := recover(); rec != nil {
				ch <- ChatResponseErr(id, ObjectChat, m.modelInfo.Name, 0, fmt.Errorf("%s", rec), Usage{})
			}

			close(ch)
		}()

		lctx, err := llama.InitFromModel(m.model, m.ctxParams)
		if err != nil {
			ch <- ChatResponseErr(id, ObjectChat, m.modelInfo.Name, 0, fmt.Errorf("unable to init from model: %w", err), Usage{})
			return
		}

		defer func() {
			llama.Synchronize(lctx)
			llama.Free(lctx)
		}()

		prompt, err := m.applyChatRequestJinjaTemplate(cr, true)
		if err != nil {
			ch <- ChatResponseErr(id, ObjectChat, m.modelInfo.Name, 0, fmt.Errorf("unable to apply jinja template: %w", err), Usage{})
			return
		}

		m.processTokens(ctx, id, lctx, ObjectChat, prompt, cr.Params, ch)
	}()

	return ch
}
