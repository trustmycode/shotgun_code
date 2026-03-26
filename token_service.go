package main

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"sync"
	"unicode/utf8"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

type TokenEstimate struct {
	Tokens int    `json:"tokens"`
	Method string `json:"method"`
	Model  string `json:"model"`
}

type TokenService struct {
	mu    sync.RWMutex
	cache map[string]TokenEstimate
}

func NewTokenService() *TokenService {
	return &TokenService{cache: make(map[string]TokenEstimate)}
}

func (s *TokenService) Estimate(providerName, model, text string) TokenEstimate {
	normalizedProvider := normalizeProviderName(providerName)
	normalizedModel := normalizeTokenModel(normalizedProvider, model)
	key := tokenCacheKey(normalizedProvider, normalizedModel, text)

	s.mu.RLock()
	if cached, ok := s.cache[key]; ok {
		s.mu.RUnlock()
		return cached
	}
	s.mu.RUnlock()

	estimate := TokenEstimate{Model: normalizedModel}
	switch normalizedProvider {
	case LLMProviderOpenAI, LLMProviderOpenRouter:
		estimate = estimateWithTikToken(normalizedModel, text)
		if estimate.Model == "" {
			estimate.Model = normalizedModel
		}
	default:
		estimate = heuristicEstimate(normalizedModel, text)
	}

	s.mu.Lock()
	s.cache[key] = estimate
	s.mu.Unlock()
	return estimate
}

func estimateWithTikToken(model, text string) TokenEstimate {
	if strings.TrimSpace(text) == "" {
		return TokenEstimate{Tokens: 0, Method: "tiktoken", Model: model}
	}

	enc, err := tiktoken.EncodingForModel(model)
	if err != nil {
		enc, err = tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			return heuristicEstimate(model, text)
		}
	}
	tokens := len(enc.Encode(text, nil, nil))
	return TokenEstimate{Tokens: tokens, Method: "tiktoken", Model: model}
}

func heuristicEstimate(model, text string) TokenEstimate {
	runeCount := utf8.RuneCountInString(text)
	if runeCount == 0 {
		return TokenEstimate{Tokens: 0, Method: "heuristic", Model: model}
	}
	tokens := runeCount / 4
	if runeCount%4 != 0 {
		tokens++
	}
	if tokens < 1 {
		tokens = 1
	}
	return TokenEstimate{Tokens: tokens, Method: "heuristic", Model: model}
}

func normalizeTokenModel(providerName, model string) string {
	m := strings.TrimSpace(model)
	if m == "" {
		return ""
	}
	m = strings.ToLower(m)
	if providerName == LLMProviderOpenRouter {
		if idx := strings.IndexByte(m, '/'); idx > 0 && idx+1 < len(m) {
			return m[idx+1:]
		}
	}
	return m
}

func tokenCacheKey(providerName, model, text string) string {
	h := sha1.Sum([]byte(text))
	return providerName + "|" + model + "|" + hex.EncodeToString(h[:])
}

func (a *App) EstimateTokens(providerName, model, text string) (TokenEstimate, error) {
	if a.tokenService == nil {
		a.tokenService = NewTokenService()
	}

	providerName = normalizeProviderName(providerName)
	if providerName == "" {
		providerName = normalizeProviderName(a.settings.LLMSettings.ActiveProvider)
	}
	if strings.TrimSpace(model) == "" {
		model = fallbackModel(a.settings.LLMSettings)
	}

	return a.tokenService.Estimate(providerName, model, text), nil
}
