package llm

import (
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

// ChatModel is the unified model interface that accepts explicit requests and returns unified responses
type ChatModel interface {
	Generate(ctx runtime.Context, req *Request) (*Response, error)
	GenerateStream(ctx runtime.Context, req *Request) (<-chan schema.StreamEvent, error)
	SupportsTools() bool
	SupportsStreaming() bool
	Info() ModelInfo
}

// Request encapsulates a single generation request
type Request struct {
	Messages       []schema.Message       `json:"messages"`
	Config         *GenerationConfig      `json:"config,omitempty"`
	Tools          []ToolSpec             `json:"tools,omitempty"`
	ToolChoice     *ToolChoiceOption      `json:"tool_choice,omitempty"`
	ResponseFormat *ResponseFormat        `json:"response_format,omitempty"`
	Extra          map[string]interface{} `json:"extra,omitempty"`
}

// Response encapsulates model output and metadata
type Response struct {
	Message      schema.Message `json:"message"`
	Usage        TokenUsage     `json:"usage"`
	FinishReason string         `json:"finish_reason"`
	ModelInfo    ModelInfo      `json:"model_info"`
}

// ToolSpec describes a functional tool that can be called by the model
type ToolSpec struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ToolChoiceOption describes the tool selection strategy
// Type: auto/none/required/function; when set to function, use Name to specify the function name
type ToolChoiceOption struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// ResponseFormat structured output format (aligned with litellm)
type ResponseFormat struct {
	Type       string      `json:"type"` // text, json_object, json_schema
	JSONSchema interface{} `json:"json_schema,omitempty"`
	Strict     *bool       `json:"strict,omitempty"`
}

// ModelInfo basic model information
type ModelInfo struct {
	Name         string   `json:"name"`
	Provider     string   `json:"provider"`
	Version      string   `json:"version"`
	MaxTokens    int      `json:"max_tokens"`
	ContextSize  int      `json:"context_size"`
	Capabilities []string `json:"capabilities"`
}

// ModelCapability capability identifier
type ModelCapability string

const (
	CapabilityChat         ModelCapability = "chat"
	CapabilityCompletion   ModelCapability = "completion"
	CapabilityToolCalling  ModelCapability = "tool_calling"
	CapabilityStreaming    ModelCapability = "streaming"
	CapabilityMultimodal   ModelCapability = "multimodal"
	CapabilityFunctionCall ModelCapability = "function_call"
)

// GenerationConfig sampling and length control
type GenerationConfig struct {
	Temperature      float64  `json:"temperature"`
	TopP             float64  `json:"top_p"`
	TopK             int      `json:"top_k"`
	MaxTokens        int      `json:"max_tokens"`
	StopSequences    []string `json:"stop_sequences"`
	PresencePenalty  float64  `json:"presence_penalty"`
	FrequencyPenalty float64  `json:"frequency_penalty"`
	Seed             *int64   `json:"seed"`
}

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

// BaseModel provides common implementation
type BaseModel struct {
	info   ModelInfo
	config *GenerationConfig
}

func NewBaseModel(info ModelInfo, config *GenerationConfig) *BaseModel {
	if config == nil {
		config = DefaultGenerationConfig
	}
	return &BaseModel{info: info, config: config}
}

func (m *BaseModel) Info() ModelInfo                    { return m.info }
func (m *BaseModel) GetConfig() *GenerationConfig       { return m.config }
func (m *BaseModel) SetConfig(config *GenerationConfig) { m.config = config }

func (m *BaseModel) SupportsCapability(capability ModelCapability) bool {
	for _, c := range m.info.Capabilities {
		if c == string(capability) {
			return true
		}
	}
	return false
}
func (m *BaseModel) SupportsTools() bool {
	return m.SupportsCapability(CapabilityToolCalling) || m.SupportsCapability(CapabilityFunctionCall)
}
func (m *BaseModel) SupportsStreaming() bool { return m.SupportsCapability(CapabilityStreaming) }

// TokenUsage statistics
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
