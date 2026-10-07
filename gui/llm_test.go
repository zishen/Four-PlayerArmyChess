package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAIConfigReady(t *testing.T) {
	if (aiConfig{}).ready() {
		t.Fatal("empty config should not be ready")
	}
	if (aiConfig{APIURL: "http://x"}).ready() {
		t.Fatal("config without a key should not be ready")
	}
	if !(aiConfig{APIURL: "http://x", APIKey: "k"}).ready() {
		t.Fatal("config with url and key should be ready")
	}
}

func TestDefaultConfigPathAndRoundTrip(t *testing.T) {
	if defaultAIConfig().Model == "" {
		t.Fatal("default model should be set")
	}
	if !strings.HasSuffix(aiConfigPath(), "ai_config.json") {
		t.Fatalf("unexpected config path %q", aiConfigPath())
	}
	in := aiConfig{APIURL: "http://example/v1", APIKey: "secret", Model: "m1"}
	if err := saveAIConfig(in); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(aiConfigPath())
	got := loadAIConfig()
	if got.APIURL != in.APIURL || got.APIKey != in.APIKey || got.Model != in.Model {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestLoadAIConfigEnvOverride(t *testing.T) {
	t.Setenv("FOURJUN_LLM_URL", "http://env/v1")
	t.Setenv("FOURJUN_LLM_KEY", "envkey")
	t.Setenv("FOURJUN_LLM_MODEL", "envmodel")
	cfg := loadAIConfig()
	if cfg.APIURL != "http://env/v1" || cfg.APIKey != "envkey" || cfg.Model != "envmodel" {
		t.Fatalf("env override failed: %+v", cfg)
	}
}

func TestRequestLLMMoveNotConfigured(t *testing.T) {
	if _, err := requestLLMMove(aiConfig{}, "hi"); err == nil {
		t.Fatal("unconfigured request should error")
	}
}

func TestRequestLLMMoveSuccessAndHTTPError(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "7"}}},
		})
	}))
	defer ok.Close()
	out, err := requestLLMMove(aiConfig{APIURL: ok.URL, APIKey: "key", Model: "m"}, "prompt")
	if err != nil || out != "7" {
		t.Fatalf("got %q err %v", out, err)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer bad.Close()
	if _, err := requestLLMMove(aiConfig{APIURL: bad.URL, APIKey: "k", Model: "m"}, "p"); err == nil {
		t.Fatal("non-200 should error")
	}

	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer empty.Close()
	if _, err := requestLLMMove(aiConfig{APIURL: empty.URL, APIKey: "k", Model: "m"}, "p"); err == nil {
		t.Fatal("empty choices should error")
	}

	garbage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer garbage.Close()
	if _, err := requestLLMMove(aiConfig{APIURL: garbage.URL, APIKey: "k", Model: "m"}, "p"); err == nil {
		t.Fatal("malformed json should error")
	}
}

func TestRequestLLMMoveEndpointSuffix(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Path
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"1"}}]}`))
	}))
	defer srv.Close()
	_, _ = requestLLMMove(aiConfig{APIURL: srv.URL, APIKey: "k", Model: "m"}, "p")
	if seen != "/chat/completions" {
		t.Fatalf("expected /chat/completions path, got %q", seen)
	}
}
