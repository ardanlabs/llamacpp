package kronk

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hybridgroup/yzma/pkg/llama"
)

func (m *model) chat(ctx context.Context, messages []ChatMessage, params Params) (ChatResponse, error) {
	ch := m.chatStreaming(ctx, messages, params)

	var lastMsg ChatResponse
	for msg := range ch {
		lastMsg = msg
	}

	return lastMsg, nil
}

func (m *model) chatStreaming(ctx context.Context, messages []ChatMessage, params Params) <-chan ChatResponse {
	ch := make(chan ChatResponse)

	go func() {
		id := uuid.New().String()

		defer func() {
			if rec := recover(); rec != nil {
				ch <- chatResponseErr(id, ObjectChat, m.modelName, 0, fmt.Errorf("%s", rec), Usage{})
			}

			close(ch)
		}()

		lctx, err := llama.InitFromModel(m.model, m.ctxParams)
		if err != nil {
			ch <- chatResponseErr(id, ObjectChat, m.modelName, 0, fmt.Errorf("unable to init from model: %w", err), Usage{})
			return
		}

		defer func() {
			llama.Synchronize(lctx)
			llama.Free(lctx)
		}()

		prompt := m.applyChatTemplate(messages)
		m.processTokens(ctx, id, lctx, ObjectChat, prompt, params, ch)
	}()

	return ch
}

func (m *model) applyChatTemplate(messages []ChatMessage) string {
	msgs := make([]llama.ChatMessage, len(messages))
	for i, msg := range messages {
		msgs[i] = llama.NewChatMessage(msg.Role, msg.Content)
	}

	buf := make([]byte, m.cfg.ContextWindow)
	l := llama.ChatApplyTemplate(m.template, msgs, true, buf)

	return string(buf[:l])
}
