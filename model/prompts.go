package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/hybridgroup/yzma/pkg/llama"
	"github.com/hybridgroup/yzma/pkg/mtmd"
	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/nikolalohinski/gonja/v2/loaders"
)

func (m *Model) applyChatRequestJinjaTemplate(cr ChatRequest, addAssistantPrompt bool) (string, error) {
	t, err := m.applyJinjaTemplate(cr.Messages, addAssistantPrompt)
	if err != nil {
		if m.modelInfo.IsGPT {
			return m.applyDefaultTemplate(cr.Messages), nil
		}

		return "", err
	}

	return t, nil
}

func (m *Model) applyVisionRequestJinjaTemplate(vr VisionRequest, addAssistantPrompt bool) (string, error) {
	messages := []ChatMessage{
		vr.Message,
		{
			Role:    "user",
			Content: mtmd.DefaultMarker(),
		},
	}

	t, err := m.applyJinjaTemplate(messages, addAssistantPrompt)
	if err != nil {
		if m.modelInfo.IsGPT {
			return m.applyDefaultTemplate(messages), nil
		}

		return "", err
	}

	return t, nil
}

func (m *Model) applyJinjaTemplate(messages []ChatMessage, addAssistantPrompt bool) (string, error) {
	if m.template == "" {
		return "", errors.New("no template found")
	}

	gonja.DefaultLoader = &noFSLoader{}

	t, err := gonja.FromString(m.template)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	jsonData, err := json.Marshal(messages)
	if err != nil {
		return "", fmt.Errorf("failed to marshal messages: %w", err)
	}

	var obj []map[string]any
	if err := json.Unmarshal(jsonData, &obj); err != nil {
		return "", fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	data := exec.NewContext(map[string]any{
		"messages":              obj,
		"add_generation_prompt": addAssistantPrompt,
	})

	s, err := t.ExecuteToString(data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return s, nil
}

func (m *Model) applyDefaultTemplate(messages []ChatMessage) string {
	msgs := make([]llama.ChatMessage, len(messages))
	for i, msg := range messages {
		msgs[i] = llama.NewChatMessage(msg.Role, msg.Content)
	}

	buf := make([]byte, m.cfg.ContextWindow)
	l := llama.ChatApplyTemplate(m.template, msgs, true, buf)

	return string(buf[:l])
}

// =============================================================================

type noFSLoader struct{}

func (nl *noFSLoader) Read(path string) (io.Reader, error) {
	return nil, errors.New("filesystem access disabled")
}

func (nl *noFSLoader) Resolve(path string) (string, error) {
	return "", errors.New("filesystem access disabled")
}

func (nl *noFSLoader) Inherit(from string) (loaders.Loader, error) {
	return nil, errors.New("filesystem access disabled")
}

// =============================================================================

func readJinjaTemplate(fileName string) (string, error) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), nil
}
