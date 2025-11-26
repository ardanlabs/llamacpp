package kronk_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ardanlabs/kronk"
	"github.com/ardanlabs/kronk/model"
	"golang.org/x/sync/errgroup"
)

func Test_Embedding(t *testing.T) {
	// Run on all platforms.
	testEmbedding(t, modelEmbedFile)
}

func testEmbedding(t *testing.T, modelFile string) {
	cfg := model.Config{
		Embeddings: true,
	}

	krn, err := kronk.New(concurrency, modelFile, "", cfg)
	if err != nil {
		t.Fatalf("unable to create inference model: %v", err)
	}
	defer krn.Unload()

	// -------------------------------------------------------------------------

	text := "Embed this sentence"

	f := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*5*time.Second)
		defer cancel()

		embed, err := krn.Embed(ctx, text)
		if err != nil {
			return fmt.Errorf("embed: %w", err)
		}

		if embed[0] == 0 || embed[len(embed)-1] == 0 {
			return fmt.Errorf("expected to have values in the embedding")
		}

		return nil
	}

	var g errgroup.Group
	for range concurrency {
		g.Go(f)
	}

	if err := g.Wait(); err != nil {
		t.Errorf("error: %v", err)
	}
}
