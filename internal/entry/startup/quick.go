package startup

import (
	"fmt"
	"os"
	"strings"

	"github.com/voocel/ainovel-cli/internal/i18n"
)

// LoadPromptFile 读取文件作为初始创作要求。
func LoadPromptFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf(i18n.T("startup.err.promptfile"), err)
	}
	return strings.TrimSpace(string(data)), nil
}

// PrepareQuick 整理快速启动提示词。
func PrepareQuick(rawPrompt string) (string, error) {
	prompt := strings.TrimSpace(rawPrompt)
	if prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}
	return prompt, nil
}
