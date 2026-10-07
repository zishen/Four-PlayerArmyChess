package gui

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// aiConfig holds the settings for the optional large language model opponent.
type aiConfig struct {
	APIURL string `json:"api_url"`
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

func aiConfigPath() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), "ai_config.json")
	}
	return "ai_config.json"
}

func defaultAIConfig() aiConfig {
	return aiConfig{Model: "gpt-4o-mini"}
}

// loadAIConfig reads ai_config.json next to the executable. Environment
// variables override the file so the key need not be written to disk.
func loadAIConfig() aiConfig {
	cfg := defaultAIConfig()
	if data, err := os.ReadFile(aiConfigPath()); err == nil {
		_ = json.Unmarshal(data, &cfg)
	}
	if cfg.Model == "" {
		cfg.Model = defaultAIConfig().Model
	}
	if v := strings.TrimSpace(os.Getenv("FOURJUN_LLM_URL")); v != "" {
		cfg.APIURL = v
	}
	if v := strings.TrimSpace(os.Getenv("FOURJUN_LLM_KEY")); v != "" {
		cfg.APIKey = v
	}
	if v := strings.TrimSpace(os.Getenv("FOURJUN_LLM_MODEL")); v != "" {
		cfg.Model = v
	}
	return cfg
}

func saveAIConfig(cfg aiConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(aiConfigPath(), data, 0600)
}

func (c aiConfig) ready() bool {
	return strings.TrimSpace(c.APIURL) != "" && strings.TrimSpace(c.APIKey) != ""
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// requestLLMMove sends an OpenAI-compatible chat completion request and
// returns the assistant text.
func requestLLMMove(cfg aiConfig, prompt string) (string, error) {
	if !cfg.ready() {
		return "", errors.New("未配置大模型接口地址或 API Key")
	}
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.APIURL), "/")
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}
	payload, err := json.Marshal(chatRequest{
		Model: cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: "你是四国军棋专家。只输出所选走法的编号，不要输出解释。"},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("大模型返回状态 %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("大模型没有返回内容")
	}
	return parsed.Choices[0].Message.Content, nil
}

// showAISettingsDialog lets the user edit and persist the LLM settings.
func showAISettingsDialog(owner walk.Form, cfg *aiConfig) {
	var dlg *walk.Dialog
	var urlEdit, keyEdit, modelEdit *walk.LineEdit
	var okBtn, cancelBtn *walk.PushButton
	_, _ = Dialog{
		AssignTo:      &dlg,
		Title:         "大模型设置",
		DefaultButton: &okBtn,
		CancelButton:  &cancelBtn,
		MinSize:       Size{Width: 520, Height: 240},
		Layout:        VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}},
		Children: []Widget{
			Label{Text: "接口地址（OpenAI 兼容，如 https://api.openai.com/v1）："},
			LineEdit{AssignTo: &urlEdit, Text: cfg.APIURL, CueBanner: "https://api.openai.com/v1"},
			Label{Text: "API Key：", MinSize: Size{Height: 22}},
			LineEdit{AssignTo: &keyEdit, Text: cfg.APIKey, PasswordMode: true},
			Label{Text: "模型名称："},
			LineEdit{AssignTo: &modelEdit, Text: cfg.Model, CueBanner: "gpt-4o-mini"},
			Label{Text: "提示：未配置时“大模型”难度会自动回退到高级 AI；也可用环境变量 FOURJUN_LLM_URL / FOURJUN_LLM_KEY / FOURJUN_LLM_MODEL。"},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{AssignTo: &okBtn, Text: "保存", OnClicked: func() {
					cfg.APIURL = strings.TrimSpace(urlEdit.Text())
					cfg.APIKey = strings.TrimSpace(keyEdit.Text())
					cfg.Model = strings.TrimSpace(modelEdit.Text())
					if cfg.Model == "" {
						cfg.Model = defaultAIConfig().Model
					}
					_ = saveAIConfig(*cfg)
					dlg.Accept()
				}},
				PushButton{AssignTo: &cancelBtn, Text: "取消", OnClicked: func() {
					dlg.Cancel()
				}},
			}},
		},
	}.Run(owner)
}
