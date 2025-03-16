package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenAIProvider struct {
	id           string
	apiKey       string
	baseURL      string
	defaultModel string
	httpClient   *http.Client
}

func NewOpenAIProvider(config Config) (Provider, error) {
	if config.APIKey == "" {
		return nil, ErrAPIKeyNotSet
	}

	baseURL := "https://api.openai.com/v1"
	if config.BaseURL != "" {
		baseURL = config.BaseURL
	}

	defaultModel := "gpt-4o"
	if config.DefaultModel != "" {
		defaultModel = config.DefaultModel
	}

	timeout := 30
	if config.Timeout > 0 {
		timeout = config.Timeout
	}

	return &OpenAIProvider{
		id:           "openai",
		apiKey:       config.APIKey,
		baseURL:      baseURL,
		defaultModel: defaultModel,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}, nil
}

func (p *OpenAIProvider) ID() string {
	return p.id
}

func (p *OpenAIProvider) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	if req.Model == "" {
		req.Model = p.defaultModel
	}

	reqURL := fmt.Sprintf("%s/chat/completions", p.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, ErrRequestFailed.WithDetails(err.Error())
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, ErrRequestFailed.WithDetails(err.Error())
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, ErrRequestFailed.WithDetails(err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrResponseInvalid.WithDetails(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrRequestFailed.WithDetails(fmt.Sprintf("status code: %d, body: %s", resp.StatusCode, string(body)))
	}

	var result ChatCompletionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, ErrResponseInvalid.WithDetails(err.Error())
	}

	return &result, nil
}

func (p *OpenAIProvider) GetModels(ctx context.Context) ([]string, error) {
	reqURL := fmt.Sprintf("%s/models", p.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, ErrRequestFailed.WithDetails(err.Error())
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, ErrRequestFailed.WithDetails(err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrResponseInvalid.WithDetails(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrRequestFailed.WithDetails(fmt.Sprintf("status code: %d, body: %s", resp.StatusCode, string(body)))
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, ErrResponseInvalid.WithDetails(err.Error())
	}

	models := make([]string, 0, len(result.Data))
	for _, model := range result.Data {
		models = append(models, model.ID)
	}

	return models, nil
}

func (p *OpenAIProvider) Close() error {
	return nil
}

func init() {
	factory := NewFactory()
	factory.Register("openai", func(config Config) (Provider, error) {
		return NewOpenAIProvider(config)
	})
}
