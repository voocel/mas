package llm

import "github.com/voocel/agentcore"

// ModelInfo contains basic model metadata.
type ModelInfo struct {
	Name         string   `json:"name"`
	Provider     string   `json:"provider"`
	Version      string   `json:"version"`
	MaxTokens    int      `json:"max_tokens"`
	ContextSize  int      `json:"context_size"`
	Capabilities []string `json:"capabilities"`
}

// ModelCapability defines capability identifiers.
type ModelCapability string

const (
	CapabilityChat         ModelCapability = "chat"
	CapabilityCompletion   ModelCapability = "completion"
	CapabilityToolCalling  ModelCapability = "tool_calling"
	CapabilityStreaming    ModelCapability = "streaming"
	CapabilityMultimodal   ModelCapability = "multimodal"
	CapabilityFunctionCall ModelCapability = "function_call"
)

// GenerationConfig defines sampling and length control parameters.
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

// BaseModel provides common model metadata and capability checks.
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

func (m *BaseModel) Info() ModelInfo              { return m.info }
func (m *BaseModel) GetConfig() *GenerationConfig { return m.config }

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

func (m *BaseModel) SupportsStreaming() bool {
	return m.SupportsCapability(CapabilityStreaming)
}

// Re-export root package types for convenience.
type (
	Role         = agentcore.Role
	Message      = agentcore.Message
	ContentBlock = agentcore.ContentBlock
	ContentType  = agentcore.ContentType
	ToolCall     = agentcore.ToolCall
	ToolSpec     = agentcore.ToolSpec
	LLMResponse  = agentcore.LLMResponse
	StreamEvent  = agentcore.StreamEvent
	StopReason   = agentcore.StopReason
	Usage        = agentcore.Usage
	ChatModel    = agentcore.ChatModel
	CallOption   = agentcore.CallOption
	CallConfig   = agentcore.CallConfig
)

// Re-export role constants.
var (
	RoleUser      = agentcore.RoleUser
	RoleAssistant = agentcore.RoleAssistant
	RoleSystem    = agentcore.RoleSystem
	RoleTool      = agentcore.RoleTool
)

// Re-export stream event type constants.
var (
	StreamEventTextStart     = agentcore.StreamEventTextStart
	StreamEventTextDelta     = agentcore.StreamEventTextDelta
	StreamEventTextEnd       = agentcore.StreamEventTextEnd
	StreamEventThinkingStart = agentcore.StreamEventThinkingStart
	StreamEventThinkingDelta = agentcore.StreamEventThinkingDelta
	StreamEventThinkingEnd   = agentcore.StreamEventThinkingEnd
	StreamEventToolCallStart = agentcore.StreamEventToolCallStart
	StreamEventToolCallDelta = agentcore.StreamEventToolCallDelta
	StreamEventToolCallEnd   = agentcore.StreamEventToolCallEnd
	StreamEventDone          = agentcore.StreamEventDone
	StreamEventError         = agentcore.StreamEventError
)

// Re-export content type constants.
var (
	ContentText     = agentcore.ContentText
	ContentThinking = agentcore.ContentThinking
	ContentToolCall = agentcore.ContentToolCall
	ContentImage    = agentcore.ContentImage
)
