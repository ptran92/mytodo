package utils

import (
	"os"
	"strings"
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
	return os.Getenv("USE_AI") != ""
}
