package metrics

type usageData struct {
	PromptTokens     int
	ReasoningTokens  int
	CompletionTokens int
	OutputTokens     int
	TotalTokens      int
	TokensPerSecond  float64
}

type usage struct {
	promptTokens     *avgMetric
	reasoningTokens  *avgMetric
	completionTokens *avgMetric
	outputTokens     *avgMetric
	totalTokens      *avgMetric
	tokensPerSecond  *avgMetric
}

func newUsage(name string) *usage {
	return &usage{
		promptTokens:     newAvgMetric(name + "_tokens_prompt"),
		reasoningTokens:  newAvgMetric(name + "_tokens_reasoning"),
		completionTokens: newAvgMetric(name + "_tokens_completion"),
		outputTokens:     newAvgMetric(name + "_tokens_output"),
		totalTokens:      newAvgMetric(name + "_tokens_total"),
		tokensPerSecond:  newAvgMetric(name + "_tokens_perSecond"),
	}
}

func (u *usage) add(data usageData) {
	u.promptTokens.add(int64(data.PromptTokens))
	u.reasoningTokens.add(int64(data.ReasoningTokens))
	u.completionTokens.add(int64(data.CompletionTokens))
	u.outputTokens.add(int64(data.OutputTokens))
	u.totalTokens.add(int64(data.TotalTokens))
	u.tokensPerSecond.add(int64(data.TokensPerSecond))
}
