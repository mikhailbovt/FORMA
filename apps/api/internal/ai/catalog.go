package ai

import "strings"

type Provider struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Protocol              string   `json:"protocol"`
	DefaultBaseURL        string   `json:"default_base_url"`
	DefaultModel          string   `json:"default_model"`
	SuggestedModels       []string `json:"suggested_models"`
	RequiresAPIKey        bool     `json:"requires_api_key"`
	SupportsCustomBaseURL bool     `json:"supports_custom_base_url"`
	StructuredOutput      string   `json:"structured_output"`
	Local                 bool     `json:"local"`
}

var providers = []Provider{
	{ID: "openai", Name: "OpenAI", Protocol: "openai_responses", DefaultBaseURL: "https://api.openai.com/v1", DefaultModel: "gpt-5.6-terra", SuggestedModels: []string{"gpt-5.6-terra"}, RequiresAPIKey: true, SupportsCustomBaseURL: true, StructuredOutput: "json_schema"},
	{ID: "anthropic", Name: "Anthropic", Protocol: "anthropic_messages", DefaultBaseURL: "https://api.anthropic.com/v1/messages", DefaultModel: "claude-sonnet-5", SuggestedModels: []string{"claude-sonnet-5"}, RequiresAPIKey: true, SupportsCustomBaseURL: true, StructuredOutput: "tool_schema"},
	{ID: "gemini", Name: "Google Gemini", Protocol: "gemini_generate_content", DefaultBaseURL: "https://generativelanguage.googleapis.com/v1beta", DefaultModel: "gemini-3-flash", SuggestedModels: []string{"gemini-3-flash"}, RequiresAPIKey: true, SupportsCustomBaseURL: true, StructuredOutput: "json_schema"},
	{ID: "deepseek", Name: "DeepSeek", Protocol: "openai_chat", DefaultBaseURL: "https://api.deepseek.com", DefaultModel: "deepseek-v4-flash", SuggestedModels: []string{"deepseek-v4-flash", "deepseek-v4-pro"}, RequiresAPIKey: true, SupportsCustomBaseURL: true, StructuredOutput: "json_object"},
	{ID: "qwen", Name: "Qwen / DashScope", Protocol: "openai_chat", DefaultBaseURL: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", DefaultModel: "qwen-plus", SuggestedModels: []string{"qwen-plus", "qwen-turbo", "qwen-max"}, RequiresAPIKey: true, SupportsCustomBaseURL: true, StructuredOutput: "json_object"},
	{ID: "kimi", Name: "Kimi / Moonshot", Protocol: "openai_chat", DefaultBaseURL: "https://api.moonshot.ai/v1", DefaultModel: "kimi-k3", SuggestedModels: []string{"kimi-k3", "kimi-k2.7"}, RequiresAPIKey: true, SupportsCustomBaseURL: true, StructuredOutput: "json_object"},
	{ID: "glm", Name: "GLM / Z.AI", Protocol: "openai_chat", DefaultBaseURL: "https://api.z.ai/api/paas/v4", DefaultModel: "glm-5.1", SuggestedModels: []string{"glm-5.1", "glm-5"}, RequiresAPIKey: true, SupportsCustomBaseURL: true, StructuredOutput: "json_object"},
	{ID: "custom", Name: "OpenAI-compatible", Protocol: "openai_chat", DefaultBaseURL: "", DefaultModel: "", SuggestedModels: []string{}, RequiresAPIKey: false, SupportsCustomBaseURL: true, StructuredOutput: "json_object"},
	{ID: "ollama", Name: "Ollama", Protocol: "ollama_chat", DefaultBaseURL: "http://host.docker.internal:11434/v1", DefaultModel: "qwen3", SuggestedModels: []string{"qwen3"}, RequiresAPIKey: false, SupportsCustomBaseURL: true, StructuredOutput: "json_schema", Local: true},
}

func Providers() []Provider {
	result := make([]Provider, len(providers))
	for index, provider := range providers {
		result[index] = provider
		result[index].SuggestedModels = append([]string(nil), provider.SuggestedModels...)
	}
	return result
}

func FindProvider(id string) (Provider, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, provider := range providers {
		if provider.ID == id {
			return provider, true
		}
	}
	return Provider{}, false
}
