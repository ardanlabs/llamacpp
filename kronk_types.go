package kronk

import (
	"time"
)

// Objects represent the different types of data that can be returned.
const (
	ObjectChat   = "chat"
	ObjectVision = "vision"
)

// Roles represent the different roles that can be used in a chat.
const (
	RoleAssistant = "assistant"
)

// FinishReasons represent the different reasons a response can be finished.
const (
	FinishReasonStop  = "stop"
	FinishReasonError = "error"
)

// ModelInfo represents the model's card information.
type ModelInfo struct {
	Desc        string
	Size        uint64
	HasEncoder  bool
	HasDecoder  bool
	IsRecurrent bool
	IsHybrid    bool
	Metadata    map[string]string
}

// =============================================================================

// ChatMessage represent input for chat and vision models.
type ChatMessage struct {
	Role    string
	Content string
}

// =============================================================================

// ResponseMessage represents a single message in a response.
type ResponseMessage struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Reasoning string `json:"reasoning"`
}

// Choice represents a single choice in a response.
type Choice struct {
	Index        int             `json:"index"`
	Delta        ResponseMessage `json:"delta"`
	FinishReason string          `json:"finish_reason"`
}

// Usage provides details usage information for the request.
type Usage struct {
	InputTokens      int     `json:"input_tokens"`
	ReasoningTokens  int     `json:"reasoning_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	TokensPerSecond  float64 `json:"tokens_per_second"`
}

// ChatResponse represents output for chat and vision models.
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choice  []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

func chatResponseDelta(id string, object string, model string, index int, content string, reasoning bool, u Usage) ChatResponse {
	return ChatResponse{
		ID:      id,
		Object:  object,
		Created: time.Now().UnixMilli(),
		Model:   model,
		Choice: []Choice{
			{
				Index: index,
				Delta: ResponseMessage{
					Role:      RoleAssistant,
					Content:   hasContent(content, reasoning),
					Reasoning: hasReasoning(content, reasoning),
				},
				FinishReason: "",
			},
		},
		Usage: u,
	}
}

func hasReasoning(content string, reasoning bool) string {
	if reasoning {
		return content
	}
	return ""
}

func hasContent(content string, reasoning bool) string {
	if !reasoning {
		return content
	}
	return ""
}

func chatResponseFinal(id string, object string, model string, index int, content string, reasoning string, u Usage) ChatResponse {
	return ChatResponse{
		ID:      id,
		Object:  object,
		Created: time.Now().UnixMilli(),
		Model:   model,
		Choice: []Choice{
			{
				Index: index,
				Delta: ResponseMessage{
					Role:      RoleAssistant,
					Content:   content,
					Reasoning: reasoning,
				},
				FinishReason: FinishReasonStop,
			},
		},
		Usage: u,
	}
}

func chatResponseErr(id string, object string, model string, index int, err error, u Usage) ChatResponse {
	return ChatResponse{
		ID:      id,
		Object:  object,
		Created: time.Now().UnixMilli(),
		Model:   model,
		Choice: []Choice{
			{
				Index: index,
				Delta: ResponseMessage{
					Role:    RoleAssistant,
					Content: err.Error(),
				},
				FinishReason: FinishReasonError,
			},
		},
		Usage: u,
	}
}

// =============================================================================

// RankingDocument represents input for reranking.
type RankingDocument struct {
	Document  string
	Embedding []float64
}

// Ranking represents output for reranking.
type Ranking struct {
	Document string
	Score    float64
}
