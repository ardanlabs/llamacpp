package model

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/hybridgroup/yzma/pkg/llama"
	"github.com/hybridgroup/yzma/pkg/mtmd"
)

// Vision performs a vision request and returns the final response.
func (m *Model) Vision(ctx context.Context, vr VisionRequest) (ChatResponse, error) {
	ch := m.VisionStreaming(ctx, vr)

	var lastMsg ChatResponse
	for msg := range ch {
		lastMsg = msg
	}

	return lastMsg, nil
}

// VisionStreaming performs a vision request and streams the response.
func (m *Model) VisionStreaming(ctx context.Context, vr VisionRequest) <-chan ChatResponse {
	ch := make(chan ChatResponse)

	go func() {
		id := uuid.New().String()

		defer func() {
			if rec := recover(); rec != nil {
				ch <- ChatResponseErr(id, ObjectVision, m.modelName, 0, fmt.Errorf("%s", rec), Usage{})
			}
			close(ch)
		}()

		if m.projFile == "" {
			ch <- ChatResponseErr(id, ObjectVision, m.modelName, 0, fmt.Errorf("projection file not set"), Usage{})
			return
		}

		lctx, err := llama.InitFromModel(m.model, m.ctxParams)
		if err != nil {
			ch <- ChatResponseErr(id, ObjectVision, m.modelName, 0, fmt.Errorf("unable to init from model: %w", err), Usage{})
			return
		}

		defer func() {
			llama.Synchronize(lctx)
			llama.Free(lctx)
		}()

		mctxParams := mtmd.ContextParamsDefault()
		mctxParams.UseGPU = true
		mctxParams.FlashAttentionType = llama.FlashAttentionTypeAuto

		mtmdCtx, err := mtmd.InitFromFile(m.projFile, m.model, mctxParams)
		if err != nil {
			ch <- ChatResponseErr(id, ObjectVision, m.modelName, 0, fmt.Errorf("unable to init from model: %w", err), Usage{})
			return
		}
		defer mtmd.Free(mtmdCtx)

		prompt := m.applyVisionTemplate(vr.Message)

		bitmap, err := m.processBitmap(lctx, mtmdCtx, vr.ImageFile, prompt)
		if err != nil {
			ch <- ChatResponseErr(id, ObjectVision, m.modelName, 0, err, Usage{})
			return
		}
		defer mtmd.BitmapFree(bitmap)

		m.processTokens(ctx, id, lctx, ObjectVision, prompt, vr.Params, ch)
	}()

	return ch
}

func (m *Model) applyVisionTemplate(message ChatMessage) string {
	msgs := []llama.ChatMessage{
		llama.NewChatMessage(message.Role, message.Content),
		llama.NewChatMessage("user", mtmd.DefaultMarker()),
	}

	buf := make([]byte, m.cfg.ContextWindow)
	l := llama.ChatApplyTemplate(m.template, msgs, true, buf)

	return string(buf[:l])
}

func (m *Model) processBitmap(lctx llama.Context, mtmdCtx mtmd.Context, imageFile string, prompt string) (mtmd.Bitmap, error) {
	if _, err := os.Stat(imageFile); err != nil {
		return 0, fmt.Errorf("error accessing file %q: %w", imageFile, err)
	}

	bitmap := mtmd.BitmapInitFromFile(mtmdCtx, imageFile)
	output := mtmd.InputChunksInit()
	input := mtmd.NewInputText(prompt, true, true)

	mtmd.Tokenize(mtmdCtx, output, input, []mtmd.Bitmap{bitmap})

	var n llama.Pos
	mtmd.HelperEvalChunks(mtmdCtx, lctx, output, 0, 0, int32(m.ctxParams.NBatch), true, &n)

	return bitmap, nil
}
