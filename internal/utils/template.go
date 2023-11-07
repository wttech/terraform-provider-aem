package utils

import (
	"bytes"
	"text/template"
)

func TemplateString(tplContent string, data any) (string, error) {
	tplParsed, err := template.New("string-template").Delims("[[", "]]").Parse(tplContent)
	if err != nil {
		return "", err
	}
	var tplOutput bytes.Buffer
	if err := tplParsed.Execute(&tplOutput, data); err != nil {
		return "", err
	}
	return tplOutput.String(), nil
}
