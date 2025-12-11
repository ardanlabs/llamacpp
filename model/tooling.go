package model

import (
	"encoding/json"
	"fmt"
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

func (m *Model) toolCall(lctx llama.Context, token llama.Token, sampler llama.Sampler, buf []byte) (llama.Batch, string, error) {
	var batch llama.Batch
	var content string
	var err error
	var data strings.Builder

	// Collect the content up to the location of </tool_call>.
	for {
		batch = m.nextBatch(token)
		content, token, err = m.batchResponse(lctx, batch, sampler, buf)
		if err != nil {
			return batch, "", err
		}

		if content == "</tool_call>" {
			break
		}

		data.WriteString(content)
	}

	content = strings.Trim(data.String(), "\n")
	content = fmt.Sprintf("%s\n", content)

	batch = m.nextBatch(token)

	return batch, content, nil
}

// =============================================================================

func parseToolCall(content string) []ResponseToolCall {
	var toolCalls []ResponseToolCall

	for call := range strings.SplitSeq(content, "\n") {
		toolCall := ResponseToolCall{
			ID:  uuid.NewString(),
			Raw: call,
		}

		switch {
		case len(call) == 0:
			toolCall.Status = 1
			toolCall.Error = "response missing"

		default:
			if err := json.Unmarshal([]byte(call), &toolCall); err != nil {
				toolCall.Error = err.Error()
				toolCall.Status = 2
			}
		}

		toolCalls = append(toolCalls, toolCall)
	}

	return toolCalls
}
