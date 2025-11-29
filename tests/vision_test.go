package kronk_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/ardanlabs/kronk"
	"github.com/ardanlabs/kronk/model"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

func Test_SimpleVision(t *testing.T) {
	// Run on Linux only in GitHub Actions.
	if os.Getenv("GITHUB_ACTIONS") == "true" && runtime.GOOS == "darwin" {
		t.Skip("Skipping test in GitHub Actions")
	}

	testVision(t, krnSimpleVision)
}

func Test_SimpleStreamingVision(t *testing.T) {
	// Run on Linux only in GitHub Actions.
	if os.Getenv("GITHUB_ACTIONS") == "true" && runtime.GOOS == "darwin" {
		t.Skip("Skipping test in GitHub Actions")
	}

	testVisionStreaming(t, krnSimpleVision)
}

// =============================================================================

func initVisionTest(imageFile string) model.VisionRequest {
	question := "What is in this picture?"

	vr := model.VisionRequest{
		ImageFile: imageFile,
		Message: model.ChatMessage{
			Role:    "user",
			Content: question,
		},
		Params: model.Params{
			MaxTokens: 4096,
		},
	}

	return vr
}

func testVision(t *testing.T, krn *kronk.Kronk) {
	if runInParallel {
		t.Parallel()
	}

	vr := initVisionTest(imageFile)

	f := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*5*time.Second)
		defer cancel()

		id := uuid.New().String()
		now := time.Now()
		defer func() {
			done := time.Now()
			t.Logf("%s: %s, st: %v, en: %v, Duration: %s", id, krn.ModelInfo().Name, now.Format("15:04:05.000"), done.Format("15:04:05.000"), done.Sub(now))
		}()

		resp, err := krn.Vision(ctx, vr)
		if err != nil {
			return fmt.Errorf("vision streaming: %w", err)
		}

		if err := testVisionResponse(resp, krn.ModelInfo().Name, "vision", "giraffes"); err != nil {
			return err
		}

		return nil
	}

	var g errgroup.Group
	for range goroutines {
		g.Go(f)
	}

	if err := g.Wait(); err != nil {
		t.Errorf("error: %v", err)
	}
}

func testVisionStreaming(t *testing.T, krn *kronk.Kronk) {
	if runInParallel {
		t.Parallel()
	}

	vr := initVisionTest(imageFile)

	f := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*5*time.Second)
		defer cancel()

		id := uuid.New().String()
		now := time.Now()
		defer func() {
			done := time.Now()
			t.Logf("%s: %s, st: %v, en: %v, Duration: %s", id, krn.ModelInfo().Name, now.Format("15:04:05.000"), done.Format("15:04:05.000"), done.Sub(now))
		}()

		ch, err := krn.VisionStreaming(ctx, vr)
		if err != nil {
			return fmt.Errorf("vision streaming: %w", err)
		}

		var lastResp model.ChatResponse
		for resp := range ch {
			lastResp = resp
			if err := testChatBasics(resp, krn.ModelInfo().Name, model.ObjectVision, false); err != nil {
				return err
			}
		}

		if err := testVisionResponse(lastResp, krn.ModelInfo().Name, model.ObjectVision, "giraffes"); err != nil {
			return err
		}

		return nil
	}

	var g errgroup.Group
	for range goroutines {
		g.Go(f)
	}

	if err := g.Wait(); err != nil {
		t.Errorf("error: %v", err)
	}
}
