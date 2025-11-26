package model

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hybridgroup/yzma/pkg/llama"
)

func (m *Model) Chat(ctx context.Context, cr ChatRequest) (ChatResponse, error) {
	ch := m.ChatStreaming(ctx, cr)

	var lastMsg ChatResponse
	for msg := range ch {
		lastMsg = msg
	}

	return lastMsg, nil
}

func (m *Model) ChatStreaming(ctx context.Context, cr ChatRequest) <-chan ChatResponse {
	ch := make(chan ChatResponse)

	go func() {
		id := uuid.New().String()

		defer func() {
			if rec := recover(); rec != nil {
				ch <- ChatResponseErr(id, ObjectChat, m.modelName, 0, fmt.Errorf("%s", rec), Usage{})
			}

			close(ch)
		}()

		lctx, err := llama.InitFromModel(m.model, m.ctxParams)
		if err != nil {
			ch <- ChatResponseErr(id, ObjectChat, m.modelName, 0, fmt.Errorf("unable to init from model: %w", err), Usage{})
			return
		}

		defer func() {
			llama.Synchronize(lctx)
			llama.Free(lctx)
		}()

		prompt := m.applyChatTemplate(cr)
		m.processTokens(ctx, id, lctx, ObjectChat, prompt, cr.Params, ch)
	}()

	return ch
}

func (m *Model) applyChatTemplate(cr ChatRequest) string {
	msgs := make([]llama.ChatMessage, len(cr.Messages))
	for i, msg := range cr.Messages {
		msgs[i] = llama.NewChatMessage(msg.Role, msg.Content)
	}

	buf := make([]byte, m.cfg.ContextWindow)
	l := llama.ChatApplyTemplate(m.template, msgs, true, buf)

	return string(buf[:l])
}
