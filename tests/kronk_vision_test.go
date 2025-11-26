package kronk_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ardanlabs/kronk"
	"github.com/ardanlabs/kronk/model"
	"golang.org/x/sync/errgroup"
)

func Test_SimpleVision(t *testing.T) {
	// Run on Linux only in GitHub Actions.
	if os.Getenv("GITHUB_ACTIONS") == "true" && runtime.GOOS == "darwin" {
		t.Skip("Skipping test in GitHub Actions")
	}

	testVision(t, modelSimpleVisionFile, projSimpleVisionFile)
}

func Test_SimpleVisionStreaming(t *testing.T) {
	// Run on Linux only in GitHub Actions.
	if os.Getenv("GITHUB_ACTIONS") == "true" && runtime.GOOS == "darwin" {
		t.Skip("Skipping test in GitHub Actions")
	}

	testVisionStreaming(t, modelSimpleVisionFile, projSimpleVisionFile)
}

// =============================================================================

func initVisionTest(t *testing.T, modelFile, projFile string, imageFile string) (*kronk.Kronk, model.VisionRequest) {
	krn, err := kronk.New(concurrency, modelFile, projFile, model.Config{})
	if err != nil {
		t.Fatalf("unable to create inference model: %v", err)
	}

	question := "What is in this picture?"

	vr := model.VisionRequest{
		ImageFile: imageFile,
		Message: model.ChatMessage{
			Role:    "user",
			Content: question,
		},
	}

	return krn, vr
}

func testVision(t *testing.T, modelFile string, profFile string) {
	krn, vr := initVisionTest(t, modelFile, profFile, imageFile)
	defer krn.Unload()

	f := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*5*time.Second)
		defer cancel()

		resp, err := krn.Vision(ctx, vr)
		if err != nil {
			return fmt.Errorf("vision streaming: %w", err)
		}

		if err := testChatResponse(resp, modelFile, "vision", false); err != nil {
			return err
		}

		find := "giraffes"
		if !strings.Contains(resp.Choice[0].Delta.Content, find) {
			return fmt.Errorf("expected %q, got %q", find, resp.Choice[0].Delta.Content)
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

func testVisionStreaming(t *testing.T, modelFile string, profFile string) {
	krn, vr := initVisionTest(t, modelFile, profFile, imageFile)
	defer krn.Unload()

	f := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*5*time.Second)
		defer cancel()

		ch, err := krn.VisionStreaming(ctx, vr)
		if err != nil {
			return fmt.Errorf("vision streaming: %w", err)
		}

		var lastResp model.ChatResponse
		for resp := range ch {
			if err := testChatResponse(resp, modelFile, "vision", false); err != nil {
				return err
			}

			lastResp = resp
		}

		if err := testChatResponse(lastResp, modelFile, "vision", false); err != nil {
			return err
		}

		find := "giraffes"
		if !strings.Contains(lastResp.Choice[0].Delta.Content, find) {
			return fmt.Errorf("expected %q, got %q", find, lastResp.Choice[0].Delta.Content)
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
