package utils

import (
	"os"
	"strings"
)

const (
	AIEnabledEnvVar   = "USE_AI"
	OpenAITokenEnvVar = "OPEN_AI_API_KEY"
)

func TrimResponse(s string) string {
	begin := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")

	if begin != -1 && end != -1 {
		return s[begin : end+1]
	}

	return s
}

func AgentEnabled() bool {
	return os.Getenv(AIEnabledEnvVar) != "" || os.Getenv(OpenAITokenEnvVar) != ""
}

func GetOpenAIToken() string {
	return os.Getenv(OpenAITokenEnvVar)
}
