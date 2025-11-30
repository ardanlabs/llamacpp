package model

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/hybridgroup/yzma/pkg/llama"
)

func (m *Model) thinkStart(token llama.Token, reasonFlag *int, reasonTokens *int) llama.Batch {
	*reasonFlag = 1

	batch := m.nextBatch(token)
	*reasonTokens += int(batch.NTokens)

	return batch
}

func (m *Model) thinkStop(token llama.Token, reasonFlag *int, completionTokens *int) llama.Batch {
	*reasonFlag = 0

	batch := m.nextBatch(token)
	*completionTokens += int(batch.NTokens)

	return batch
}

func (m *Model) toolCall(lctx llama.Context, token llama.Token, sampler llama.Sampler, buf []byte) (string, error) {
	var batch llama.Batch
	var content string
	var err error
	var data strings.Builder

	// Collect the content up to the location of </tool_call>.
	for {
		batch = m.nextBatch(token)
		content, token, err = m.batchResponse(lctx, batch, sampler, buf)
		if err != nil {
			return "", err
		}

		if content == "</tool_call>" {
			break
		}

		data.WriteString(content)
	}

	return data.String(), nil
}

// =============================================================================

func parseToolCall(content string) ResponseToolCall {
	// The idea is to add a unique ID to the tool call. The user
	// can use this ID to reference the tool call in the future.

	toolCall := ResponseToolCall{
		ID:  uuid.NewString(),
		Raw: content,
	}

	switch {
	case len(content) == 0:
		toolCall.Status = 1
		toolCall.Error = "response missing"

	default:
		if err := json.Unmarshal([]byte(content), &toolCall); err != nil {
			toolCall.Error = err.Error()
			toolCall.Status = 2
		}
	}

	return toolCall
}
