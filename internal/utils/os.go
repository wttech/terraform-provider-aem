package utils

import (
	"fmt"
	"strings"
)

func EnvToScript(env map[string]string) string {
	var sb strings.Builder
	sb.WriteString("#!/bin/sh\n")
	for name, value := range env {
		escapedValue := strings.ReplaceAll(value, "\"", "\\\"")
		escapedValue = strings.ReplaceAll(escapedValue, "$", "\\$")
		sb.WriteString(fmt.Sprintf("export %s=\"%s\"\n", name, escapedValue))
	}
	return sb.String()
}
