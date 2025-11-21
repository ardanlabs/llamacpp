package kronk

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/hybridgroup/yzma/pkg/llama"
	"github.com/hybridgroup/yzma/pkg/mtmd"
)

func (m *model) visionStreaming(ctx context.Context, message ChatMessage, imageFile string, params Params) <-chan ChatResponse {
	ch := make(chan ChatResponse)

	go func() {
		defer close(ch)

		if m.projFile == "" {
			ch <- ChatResponse{Err: fmt.Errorf("projection file not set")}
			return
		}

		lctx, err := llama.InitFromModel(m.model, m.ctxParams)
		if err != nil {
			ch <- ChatResponse{Err: fmt.Errorf("unable to init from model: %w", err)}
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
			ch <- ChatResponse{Err: fmt.Errorf("unable to init from model: %w", err)}
			return
		}
		defer mtmd.Free(mtmdCtx)

		prompt := m.applyVisionTemplate(message)

		bitmap, err := m.processBitmap(lctx, mtmdCtx, imageFile, prompt)
		if err != nil {
			ch <- ChatResponse{Err: err}
			return
		}
		defer mtmd.BitmapFree(bitmap)

		m.processVisionStreaming(ctx, lctx, toSampler(params), ch)
	}()

	return ch
}

func (m *model) applyVisionTemplate(message ChatMessage) string {
	msgs := []llama.ChatMessage{
		llama.NewChatMessage(message.Role, message.Content),
		llama.NewChatMessage("user", mtmd.DefaultMarker()),
	}

	buf := make([]byte, 1024*32)
	l := llama.ChatApplyTemplate(m.template, msgs, true, buf)

	return string(buf[:l])
}

func (m *model) processBitmap(lctx llama.Context, mtmdCtx mtmd.Context, imageFile string, prompt string) (mtmd.Bitmap, error) {
	bitmap := mtmd.BitmapInitFromFile(mtmdCtx, imageFile)
	output := mtmd.InputChunksInit()
	input := mtmd.NewInputText(prompt, true, true)

	mtmd.Tokenize(mtmdCtx, output, input, []mtmd.Bitmap{bitmap})

	// Docs indicate this function is NOT thread-safe.
	func() {
		m.muHEC.Lock()
		defer m.muHEC.Unlock()
		var n llama.Pos
		mtmd.HelperEvalChunks(mtmdCtx, lctx, output, 0, 0, int32(m.ctxParams.NBatch), true, &n)
	}()

	return bitmap, nil
}

func (m *model) processVisionStreaming(ctx context.Context, lctx llama.Context, sampler llama.Sampler, ch chan<- ChatResponse) {
	batch := llama.BatchInit(1, 0, 1)
	defer llama.BatchFree(batch)

	var sz int32 = 1
	batch.NSeqId = &sz
	batch.NTokens = 1
	seqs := unsafe.SliceData([]llama.SeqId{0})
	batch.SeqId = &seqs

	buf := make([]byte, 1024*32)

	for range llama.MaxToken {
		select {
		case <-ctx.Done():
			ch <- ChatResponse{Err: ctx.Err()}
			return
		default:
		}

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

		batch = llama.BatchGetOne([]llama.Token{token})
	}
}
