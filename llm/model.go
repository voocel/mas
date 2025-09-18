package llm

import (
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

type ChatModel interface {
	Generate(ctx runtime.Context, messages []schema.Message) (schema.Message, error)

	GenerateStream(ctx runtime.Context, messages []schema.Message) (<-chan schema.StreamEvent, error)

	SupportsTools() bool

	SupportsStreaming() bool

	GetModelInfo() ModelInfo
}

type ModelInfo struct {
	Name         string   `json:"name"`
	Provider     string   `json:"provider"`
	Version      string   `json:"version"`
	MaxTokens    int      `json:"max_tokens"`
	ContextSize  int      `json:"context_size"`
	Capabilities []string `json:"capabilities"`
}

// ModelCapability defines the capabilities of a model.
type ModelCapability string

const (
	CapabilityChat         ModelCapability = "chat"
	CapabilityCompletion   ModelCapability = "completion"
	CapabilityToolCalling  ModelCapability = "tool_calling"
	CapabilityStreaming    ModelCapability = "streaming"
	CapabilityMultimodal   ModelCapability = "multimodal"
	CapabilityFunctionCall ModelCapability = "function_call"
)

// GenerationConfig is the configuration for generation.
type GenerationConfig struct {
	Temperature      float64  `json:"temperature"`       // Temperature parameter (0.0-2.0)
	TopP             float64  `json:"top_p"`             // Top-p sampling
	TopK             int      `json:"top_k"`             // Top-k sampling
	MaxTokens        int      `json:"max_tokens"`        // Maximum number of tokens to generate
	StopSequences    []string `json:"stop_sequences"`    // Stop sequences
	PresencePenalty  float64  `json:"presence_penalty"`  // Presence penalty
	FrequencyPenalty float64  `json:"frequency_penalty"` // Frequency penalty
	Seed             *int64   `json:"seed"`              // Random seed
}

// DefaultGenerationConfig is the default generation configuration.
var DefaultGenerationConfig = &GenerationConfig{
	Temperature:      0.7,
	TopP:             0.9,
	TopK:             0,
	MaxTokens:        1000,
	StopSequences:    []string{},
	PresencePenalty:  0.0,
	FrequencyPenalty: 0.0,
	Seed:             nil,
}

// BaseModel provides a base implementation for a model.
type BaseModel struct {
	info   ModelInfo
	config *GenerationConfig
}

// NewBaseModel creates a new base model.
func NewBaseModel(info ModelInfo, config *GenerationConfig) *BaseModel {
	if config == nil {
		config = DefaultGenerationConfig
	}

	return &BaseModel{
		info:   info,
		config: config,
	}
}

func (m *BaseModel) GetModelInfo() ModelInfo {
	return m.info
}

func (m *BaseModel) GetConfig() *GenerationConfig {
	return m.config
}

func (m *BaseModel) SetConfig(config *GenerationConfig) {
	m.config = config
}

// SupportsCapability checks if the model supports a specific capability.
func (m *BaseModel) SupportsCapability(capability ModelCapability) bool {
	for _, cap := range m.info.Capabilities {
		if cap == string(capability) {
			return true
		}
	}
	return false
}

func (m *BaseModel) SupportsTools() bool {
	return m.SupportsCapability(CapabilityToolCalling) || m.SupportsCapability(CapabilityFunctionCall)
}

func (m *BaseModel) SupportsStreaming() bool {
	return m.SupportsCapability(CapabilityStreaming)
}

// Generate is a base implementation that needs to be overridden by subclasses.
func (m *BaseModel) Generate(ctx runtime.Context, messages []schema.Message) (schema.Message, error) {
	return schema.Message{}, schema.NewModelError(m.info.Name, "generate", schema.ErrModelNotSupported)
}

// GenerateStream is a base implementation that needs to be overridden by subclasses.
func (m *BaseModel) GenerateStream(ctx runtime.Context, messages []schema.Message) (<-chan schema.StreamEvent, error) {
	return nil, schema.NewModelError(m.info.Name, "generate_stream", schema.ErrModelNotSupported)
}

// TokenUsage is the token usage statistics.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// GenerationResult is the result of a generation.
type GenerationResult struct {
	Message      schema.Message `json:"message"`
	Usage        TokenUsage     `json:"usage"`
	FinishReason string         `json:"finish_reason"`
	ModelInfo    ModelInfo      `json:"model_info"`
}

// FinishReason is the reason for finishing.
type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonToolCalls     FinishReason = "tool_calls"
	FinishReasonContentFilter FinishReason = "content_filter"
	FinishReasonError         FinishReason = "error"
)

// ModelRegistry is a registry for models.
type ModelRegistry struct {
	models map[string]ChatModel
}

// NewModelRegistry creates a new model registry.
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		models: make(map[string]ChatModel),
	}
}

// Register registers a model.
func (r *ModelRegistry) Register(name string, model ChatModel) error {
	if name == "" {
		return schema.NewValidationError("model.name", name, "model name cannot be empty")
	}

	if _, exists := r.models[name]; exists {
		return schema.NewModelError(name, "register", schema.ErrModelNotSupported)
	}

	r.models[name] = model
	return nil
}

// Get gets a model.
func (r *ModelRegistry) Get(name string) (ChatModel, bool) {
	model, exists := r.models[name]
	return model, exists
}

// List lists all models.
func (r *ModelRegistry) List() map[string]ChatModel {
	result := make(map[string]ChatModel)
	for name, model := range r.models {
		result[name] = model
	}
	return result
}

// Names gets all model names.
func (r *ModelRegistry) Names() []string {
	names := make([]string, 0, len(r.models))
	for name := range r.models {
		names = append(names, name)
	}
	return names
}

// Global model registry.
var globalModelRegistry = NewModelRegistry()

// GlobalModelRegistry gets the global model registry.
func GlobalModelRegistry() *ModelRegistry {
	return globalModelRegistry
}

// RegisterModel registers a model in the global registry.
func RegisterModel(name string, model ChatModel) error {
	return globalModelRegistry.Register(name, model)
}

// GetModel gets a model from the global registry.
func GetModel(name string) (ChatModel, bool) {
	return globalModelRegistry.Get(name)
}

// ListModels lists all models in the global registry.
func ListModels() map[string]ChatModel {
	return globalModelRegistry.List()
}

// ModelNames gets all model names in the global registry.
func ModelNames() []string {
	return globalModelRegistry.Names()
}
