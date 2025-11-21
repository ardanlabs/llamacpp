// Package kronk provides support for working with models using llamacpp via yzma.
package kronk

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hybridgroup/yzma/pkg/llama"
	"github.com/hybridgroup/yzma/pkg/mtmd"
)

// LogLevel represents the logging level.
type LogLevel int

// Set of logging levels supported by llamacpp.
const (
	LogSilent LogLevel = iota + 1
	LogNormal
)

var (
	libraryLocation string
	initOnce        sync.Once
	initErr         error
)

// Init initializes the llamacpp and yzma libraries.
func Init(libPath string, logLevel LogLevel) error {
	initOnce.Do(func() {
		if err := llama.Load(libPath); err != nil {
			initErr = fmt.Errorf("unable to load library: %w", err)
			return
		}

		libraryLocation = libPath

		llama.Init()

		switch logLevel {
		case LogSilent:
			llama.LogSet(llama.LogSilent())
		default:
			llama.LogSet(llama.LogNormal)
		}
	})

	return initErr
}

// Llama represents a concurrency group of a specified model.
type Llama struct {
	modelName string
	llama     chan *model
	wg        sync.WaitGroup
	closed    uint32
}

// New provides the ability to use models in a concurrently safe way.
func New(concurrency int, modelFile string, cfg Config, options ...func(llg *model) error) (*Llama, error) {
	if libraryLocation == "" {
		return nil, fmt.Errorf("the Init() function has not been called")
	}

	if concurrency <= 0 {
		return nil, fmt.Errorf("concurrency must be > 0, got %d", concurrency)
	}

	ch := make(chan *model, concurrency)

	for range concurrency {
		l, err := newModel(modelFile, cfg, options...)
		if err != nil {
			close(ch)
			for m := range ch {
				m.unload()
			}

			return nil, err
		}

		ch <- l
	}

	mgr := Llama{
		modelName: strings.TrimSuffix(filepath.Base(modelFile), filepath.Ext(modelFile)),
		llama:     ch,
	}

	return &mgr, nil
}

func WithProjection(projFile string) func(m *model) error {
	return func(m *model) error {
		if err := mtmd.Load(libraryLocation); err != nil {
			return fmt.Errorf("unable to load mtmd library: %w", err)
		}

		m.projFile = projFile

		return nil
	}
}

// ModelName returns the model name.
func (llm *Llama) ModelName() string {
	return llm.modelName
}

// Unload will close down all loaded models. You should call this only when you
// are completely done using the group.
func (llm *Llama) Unload() {
	if !atomic.CompareAndSwapUint32(&llm.closed, 0, 1) {
		return
	}

	llm.wg.Wait()

	close(llm.llama)
	for llama := range llm.llama {
		llama.unload()
	}
}

// ModelInfo provides support to extract the model card information.
func (llm *Llama) ModelInfo(ctx context.Context) (ModelInfo, error) {
	if atomic.LoadUint32(&llm.closed) == 1 {
		return ModelInfo{}, fmt.Errorf("Llama has been unloaded")
	}

	select {
	case <-ctx.Done():
		return ModelInfo{}, ctx.Err()

	case llama, ok := <-llm.llama:
		if !ok {
			return ModelInfo{}, fmt.Errorf("Llama has been unloaded")
		}

		llm.wg.Add(1)
		defer func() {
			llm.llama <- llama
			llm.wg.Done()
		}()

		return llama.modelInfo(), nil
	}
}

// ChatCompletions provides support to interact with an inference model.
// It will block until a model becomes available or the context times out.
func (llm *Llama) ChatCompletions(ctx context.Context, messages []ChatMessage, params Params) (<-chan ChatResponse, error) {
	if atomic.LoadUint32(&llm.closed) == 1 {
		return nil, fmt.Errorf("Llama has been unloaded")
	}

	ch := make(chan ChatResponse)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case llama, ok := <-llm.llama:
		if !ok {
			return nil, fmt.Errorf("Llama has been unloaded")
		}

		llm.wg.Add(1)
		go func() {
			defer func() {
				close(ch)
				llm.llama <- llama
				llm.wg.Done()
			}()

			lch := llama.chatCompletions(ctx, messages, params)
			for msg := range lch {
				ch <- msg
			}
		}()
	}

	return ch, nil
}

// ChatVision provides support to interact with a vision language model. It will
// block until a model becomes available or the context times out.
func (llm *Llama) ChatVision(ctx context.Context, message ChatMessage, imageFile string, params Params) (<-chan ChatResponse, error) {
	if atomic.LoadUint32(&llm.closed) == 1 {
		return nil, fmt.Errorf("Llama has been unloaded")
	}

	ch := make(chan ChatResponse)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case llama, ok := <-llm.llama:
		if !ok {
			return nil, fmt.Errorf("Llama has been unloaded")
		}

		llm.wg.Add(1)
		go func() {
			defer func() {
				close(ch)
				llm.llama <- llama
				llm.wg.Done()
			}()

			lch := llama.chatVision(ctx, message, imageFile, params)
			for msg := range lch {
				ch <- msg
			}
		}()
	}

	return ch, nil
}

// Embed provides support to interact with an embedding model. It will block
// until a model becomes available or the context times out.
func (llm *Llama) Embed(ctx context.Context, text string) ([]float32, error) {
	if atomic.LoadUint32(&llm.closed) == 1 {
		return nil, fmt.Errorf("Llama has been unloaded")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case llama, ok := <-llm.llama:
		if !ok {
			return nil, fmt.Errorf("Llama has been unloaded")
		}

		llm.wg.Add(1)
		defer func() {
			llm.llama <- llama
			llm.wg.Done()
		}()

		vec, err := llama.embed(ctx, text)
		if err != nil {
			return nil, err
		}

		return vec, nil
	}
}

// Rerank provides support to rerank a set of embeddings.
func (llm *Llama) Rerank(rankingDocs []RankingDocument) ([]Ranking, error) {
	rerankedDocs := make([]Ranking, len(rankingDocs))

	// Simple scoring based on embedding magnitude and positive values.
	for i, doc := range rankingDocs {
		if len(doc.Embedding) == 0 {
			rerankedDocs[i] = Ranking{Document: doc.Document, Score: 0}
			continue
		}

		var sumPositive, sumTotal float64
		for _, val := range doc.Embedding {
			sumTotal += val * val
			if val > 0 {
				sumPositive += val
			}
		}

		if sumTotal == 0 {
			rerankedDocs[i] = Ranking{Document: doc.Document, Score: 0}
			continue
		}

		// Normalize and combine magnitude with positive bias
		magnitude := math.Sqrt(sumTotal) / float64(len(doc.Embedding))
		positiveRatio := sumPositive / float64(len(doc.Embedding))
		score := (magnitude + positiveRatio) / 2

		rerankedDocs[i] = Ranking{Document: doc.Document, Score: score}
	}

	sort.Slice(rerankedDocs, func(i, j int) bool {
		return rerankedDocs[i].Score > rerankedDocs[j].Score
	})

	return rerankedDocs, nil
}
