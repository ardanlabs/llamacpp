// Package metrics constructs the metrics the application will track.
package metrics

import (
	"expvar"
	"runtime"
	"time"
)

var m metrics

type metrics struct {
	goroutines          *expvar.Int
	requests            *expvar.Int
	errors              *expvar.Int
	panics              *expvar.Int
	modelFileLoadTime   *avgMetric
	projFileLoadTime    *avgMetric
	promptCreationTime  *avgMetric
	prefillNonMediaTime *avgMetric
	prefillMediaTime    *avgMetric
	timeToFirstToken    *avgMetric
	chatCompletions     *usage
}

func init() {
	m = metrics{
		goroutines:          expvar.NewInt("service_goroutines"),
		requests:            expvar.NewInt("service_requests"),
		errors:              expvar.NewInt("service_errors"),
		panics:              expvar.NewInt("service_panics"),
		modelFileLoadTime:   newAvgMetric("file_modelLoadTime"),
		projFileLoadTime:    newAvgMetric("file_projLoadTime"),
		promptCreationTime:  newAvgMetric("prompt_creationTime"),
		prefillNonMediaTime: newAvgMetric("prefill_nonMediaTime"),
		prefillMediaTime:    newAvgMetric("prefill_mediaTime"),
		timeToFirstToken:    newAvgMetric("process_ttft"),
		chatCompletions:     newUsage("usage_chatCompletions"),
	}
}

// AddGoroutines refreshes the goroutine metric.
func AddGoroutines() int64 {
	g := int64(runtime.NumGoroutine())
	m.goroutines.Set(g)
	return g
}

// AddRequests increments the request metric by 1.
func AddRequests() int64 {
	m.requests.Add(1)
	return m.requests.Value()
}

// AddErrors increments the errors metric by 1.
func AddErrors() int64 {
	m.errors.Add(1)
	return m.errors.Value()
}

// AddPanics increments the panics metric by 1.
func AddPanics() int64 {
	m.panics.Add(1)
	return m.panics.Value()
}

// AddModelFileLoadTime captures the specified duration for loading a model file.
func AddModelFileLoadTime(duration time.Duration) {
	m.modelFileLoadTime.add(duration.Milliseconds())
}

// AddProjFileLoadTime captures the specified duration for loading a proj file.
func AddProjFileLoadTime(duration time.Duration) {
	m.projFileLoadTime.add(duration.Milliseconds())
}

// AddPromptCreationTime captures the specified duration for creating a prompt.
func AddPromptCreationTime(duration time.Duration) {
	m.promptCreationTime.add(duration.Milliseconds())
}

// AddPrefillNonMediaTime captures the specified duration for prefilling a non media call.
func AddPrefillNonMediaTime(duration time.Duration) {
	m.prefillNonMediaTime.add(duration.Milliseconds())
}

// AddPrefillMediaTime captures the specified duration for prefilling a media call.
func AddPrefillMediaTime(duration time.Duration) {
	m.prefillMediaTime.add(duration.Milliseconds())
}

// AddTimeToFirstToken captures the specified duration for ttft.
func AddTimeToFirstToken(duration time.Duration) {
	m.timeToFirstToken.add(duration.Milliseconds())
}

// AddChatCompletionsUsage captures the specified usage values for chat-completions.
func AddChatCompletionsUsage(promptTokens, reasoningTokens, completionTokens, outputTokens, totalTokens int, tokensPerSecond float64) {
	data := usageData{
		PromptTokens:     promptTokens,
		ReasoningTokens:  reasoningTokens,
		CompletionTokens: completionTokens,
		OutputTokens:     outputTokens,
		TotalTokens:      totalTokens,
		TokensPerSecond:  tokensPerSecond,
	}

	m.chatCompletions.add(data)
}
