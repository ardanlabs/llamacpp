package model

import (
	"fmt"
	"strconv"

	"github.com/hybridgroup/yzma/pkg/llama"
)

const (
	defTopK      = 40
	defTopP      = 0.9
	defTemp      = 0.7
	defMaxTokens = 512
)

// Params represents the different options when using a model. The defaults are
// used when these values are set to 0.
//
// Temperature controls the randomness of the output. It rescales the probability
// distribution of possible next tokens.
// When set to 0, the default value is 0.7.
//
// TopK limits the pool of possible next tokens to the K number of most probable
// tokens. If a model predicts 10,000 possible next tokens, setting Top-K to 50
// means only the 50 tokens with the highest probabilities are considered for
// selection (after temperature scaling). The rest are ignored.
// When set to 0, the default value is 40.
//
// TopP, also known as nucleus sampling, works differently than Top-K by
// selecting a dynamic pool of tokens whose cumulative probability exceeds a
// threshold P. Instead of a fixed number of tokens (K), it selects the minimum
// number of most probable tokens required to reach the cumulative probability P.
// When set to 0, the default value is 0.9.
//
// These parameters (TopK, TopP, Temperature) are typically used together. The
// sampling process usually applies temperature first, then filters the token
// list using Top-K, and finally filters it again using Top-P before selecting
// the next token randomly from the remaining pool based on their (now adjusted)
// probabilities.
//
// MaxTokens defines the maximum number of output tokens to generate for a
// single response.
// When set to 0, the default value is 512.
type Params struct {
	Temperature float32 `json:"temperature"`
	TopK        int32   `json:"top_k"`
	TopP        float32 `json:"top_p"`
	MaxTokens   int     `json:"max_tokens"`
}

// AddParams can be used to add the configured parameters to the
// specified document.
func AddParams(p Params, d D) {
	d["temperature"] = p.Temperature
	d["top_k"] = p.TopK
	d["top_p"] = p.TopP
	d["max_tokens"] = p.MaxTokens
}

func parseParams(d D) (Params, error) {
	var temp float32
	if tempVal, exists := d["temperature"]; exists {
		var err error
		temp, err = parseFloat32("temperature", tempVal)
		if err != nil {
			return Params{}, err
		}
	}

	var topK int
	if topKVal, exists := d["top_k"]; exists {
		var err error
		topK, err = parseInt("top_k", topKVal)
		if err != nil {
			return Params{}, err
		}
	}

	var topP float32
	if topPVal, exists := d["top_p"]; exists {
		var err error
		topP, err = parseFloat32("top_p", topPVal)
		if err != nil {
			return Params{}, err
		}
	}

	var maxTokens int
	if maxTokensVal, exists := d["max_tokens"]; exists {
		var err error
		maxTokens, err = parseInt("max_tokens", maxTokensVal)
		if err != nil {
			return Params{}, err
		}
	}

	params := Params{
		Temperature: temp,
		TopK:        int32(topK),
		TopP:        topP,
		MaxTokens:   maxTokens,
	}

	return adjustParams(params), nil
}

func adjustParams(p Params) Params {
	if p.Temperature <= 0 {
		p.Temperature = defTemp
	}

	if p.TopK <= 0 {
		p.TopK = defTopK
	}

	if p.TopP <= 0 {
		p.TopP = defTopP
	}

	if p.MaxTokens <= 0 {
		p.MaxTokens = defMaxTokens
	}

	return p
}

func toSampler(p Params) llama.Sampler {
	sampler := llama.SamplerChainInit(llama.SamplerChainDefaultParams())

	llama.SamplerChainAdd(sampler, llama.SamplerInitTempExt(p.Temperature, 0, 1.0))
	llama.SamplerChainAdd(sampler, llama.SamplerInitTopK(p.TopK))
	llama.SamplerChainAdd(sampler, llama.SamplerInitTopP(p.TopP, 0))
	llama.SamplerChainAdd(sampler, llama.SamplerInitDist(llama.DefaultSeed))

	return sampler
}

func parseFloat32(name string, val any) (float32, error) {
	var result float32

	switch v := val.(type) {
	case string:
		temp32, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return 0, fmt.Errorf("%s is not valid: %w", name, err)
		}
		result = float32(temp32)

	case float32:
		result = v

	case float64:
		result = float32(v)

	case int:
		result = float32(v)

	case int32:
		result = float32(v)

	case int64:
		result = float32(v)

	default:
		return 0, fmt.Errorf("%s is not a valid type", name)
	}

	return result, nil
}

func parseInt(name string, val any) (int, error) {
	var result int

	switch v := val.(type) {
	case string:
		temp32, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return 0, fmt.Errorf("%s is not valid: %w", name, err)
		}
		result = int(temp32)

	case float32:
		result = int(v)

	case float64:
		result = int(v)

	case int:
		result = v

	case int32:
		result = int(v)

	case int64:
		result = int(v)

	default:
		return 0, fmt.Errorf("%s is not a valid type", name)
	}

	return result, nil
}
