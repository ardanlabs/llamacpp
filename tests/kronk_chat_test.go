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

func Test_SimpleChat(t *testing.T) {
	// Run on all platforms.
	testChat(t, modelSimpleChatFile, false)
}

func Test_ThinkChat(t *testing.T) {
	// Run on Linux only in GitHub Actions.
	if os.Getenv("GITHUB_ACTIONS") == "true" && runtime.GOOS == "darwin" {
		t.Skip("Skipping test in GitHub Actions")
	}

	testChat(t, modelThinkChatFile, true)
}

func Test_GPTChat(t *testing.T) {
	// Don't run at all on GitHub Actions.
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping test in GitHub Actions")
	}

	testChat(t, modelGPTChatFile, true)
}

// =============================================================================

func Test_SimpleChatStreaming(t *testing.T) {
	// Run on all platforms.
	testChatStreaming(t, modelSimpleChatFile, false)
}

func Test_ThinkChatStreaming(t *testing.T) {
	// Run on Linux only in GitHub Actions.
	if os.Getenv("GITHUB_ACTIONS") == "true" && runtime.GOOS == "darwin" {
		t.Skip("Skipping test in GitHub Actions")
	}

	testChatStreaming(t, modelThinkChatFile, true)
}

func Test_GPTChatStreaming(t *testing.T) {
	// Don't run at all on GitHub Actions.
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping test in GitHub Actions")
	}

	testChatStreaming(t, modelGPTChatFile, true)
}

// =============================================================================

func initChatTest(t *testing.T, modelFile string) (*kronk.Kronk, model.ChatRequest) {
	krn, err := kronk.New(concurrency, modelFile, "", model.Config{})
	if err != nil {
		t.Fatalf("unable to load model: %v", err)
	}

	question := "Echo back the word: Gorilla"

	cr := model.ChatRequest{
		Messages: []model.ChatMessage{
			{Role: "user", Content: question},
		},
	}

	return krn, cr
}

func testChat(t *testing.T, modelFile string, reasoning bool) {
	krn, req := initChatTest(t, modelFile)
	defer krn.Unload()

	f := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*5*time.Second)
		defer cancel()

		resp, err := krn.Chat(ctx, req)

		if err != nil {
			return fmt.Errorf("chat streaming: %w", err)
		}

		if err := testChatResponse(resp, modelFile, "chat", reasoning); err != nil {
			return err
		}

		find := "Gorilla"
		if !strings.Contains(resp.Choice[0].Delta.Content, find) {
			return fmt.Errorf("expected %q, got %q", find, resp.Choice[0].Delta.Content)
		}

		if reasoning {
			if !strings.Contains(resp.Choice[0].Delta.Reasoning, find) {
				return fmt.Errorf("expected %q, got %q", find, resp.Choice[0].Delta.Content)
			}
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

func testChatStreaming(t *testing.T, modelFile string, reasoning bool) {
	krn, cr := initChatTest(t, modelFile)
	defer krn.Unload()

	f := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*5*time.Second)
		defer cancel()

		ch, err := krn.ChatStreaming(ctx, cr)
		if err != nil {
			return fmt.Errorf("chat streaming: %w", err)
		}

		var lastResp model.ChatResponse
		for resp := range ch {
			if err := testChatResponse(resp, modelFile, "chat", reasoning); err != nil {
				return err
			}

			lastResp = resp
		}

		if err := testChatResponse(lastResp, modelFile, "chat", reasoning); err != nil {
			return err
		}

		find := "Gorilla"
		if !strings.Contains(lastResp.Choice[0].Delta.Content, find) {
			return fmt.Errorf("expected %q, got %q", find, lastResp.Choice[0].Delta.Content)
		}

		if reasoning {
			if !strings.Contains(lastResp.Choice[0].Delta.Reasoning, find) {
				return fmt.Errorf("expected %q, got %q", find, lastResp.Choice[0].Delta.Content)
			}
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
