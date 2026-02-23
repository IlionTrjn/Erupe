package setup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestBuildDefaultConfig(t *testing.T) {
	req := FinishRequest{
		DBHost:            "myhost",
		DBPort:            5433,
		DBUser:            "myuser",
		DBPassword:        "secret",
		DBName:            "mydb",
		Host:              "10.0.0.1",
		ClientMode:        "ZZ",
		AutoCreateAccount: true,
	}
	cfg := buildDefaultConfig(req)

	// Check top-level keys from user input
	if cfg["Host"] != "10.0.0.1" {
		t.Errorf("Host = %v, want 10.0.0.1", cfg["Host"])
	}
	if cfg["ClientMode"] != "ZZ" {
		t.Errorf("ClientMode = %v, want ZZ", cfg["ClientMode"])
	}
	if cfg["AutoCreateAccount"] != true {
		t.Errorf("AutoCreateAccount = %v, want true", cfg["AutoCreateAccount"])
	}

	// Check database section
	db, ok := cfg["Database"].(map[string]interface{})
	if !ok {
		t.Fatal("Database section not a map")
	}
	if db["Host"] != "myhost" {
		t.Errorf("Database.Host = %v, want myhost", db["Host"])
	}
	if db["Port"] != 5433 {
		t.Errorf("Database.Port = %v, want 5433", db["Port"])
	}
	if db["User"] != "myuser" {
		t.Errorf("Database.User = %v, want myuser", db["User"])
	}
	if db["Password"] != "secret" {
		t.Errorf("Database.Password = %v, want secret", db["Password"])
	}
	if db["Database"] != "mydb" {
		t.Errorf("Database.Database = %v, want mydb", db["Database"])
	}

	// Check that critical sections exist
	requiredKeys := []string{
		"Host", "BinPath", "Language", "ClientMode", "Database",
		"Sign", "API", "Channel", "Entrance", "DebugOptions",
		"GameplayOptions", "Discord", "Commands", "Courses",
		"SaveDumps", "Capture", "Screenshots",
	}
	for _, key := range requiredKeys {
		if _, ok := cfg[key]; !ok {
			t.Errorf("missing required key %q", key)
		}
	}

	// Verify it marshals to valid JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	if len(data) < 100 {
		t.Errorf("config JSON unexpectedly short: %d bytes", len(data))
	}
}

func TestDetectIP(t *testing.T) {
	ws := &wizardServer{
		logger: zap.NewNop(),
		done:   make(chan struct{}),
	}
	req := httptest.NewRequest("GET", "/api/setup/detect-ip", nil)
	w := httptest.NewRecorder()
	ws.handleDetectIP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	ip, ok := resp["ip"]
	if !ok || ip == "" {
		t.Error("expected non-empty IP in response")
	}
}

func TestClientModes(t *testing.T) {
	ws := &wizardServer{
		logger: zap.NewNop(),
		done:   make(chan struct{}),
	}
	req := httptest.NewRequest("GET", "/api/setup/client-modes", nil)
	w := httptest.NewRecorder()
	ws.handleClientModes(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp map[string][]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	modes := resp["modes"]
	if len(modes) != 41 {
		t.Errorf("got %d modes, want 41", len(modes))
	}
	// First should be S1.0, last should be ZZ
	if modes[0] != "S1.0" {
		t.Errorf("first mode = %q, want S1.0", modes[0])
	}
	if modes[len(modes)-1] != "ZZ" {
		t.Errorf("last mode = %q, want ZZ", modes[len(modes)-1])
	}
}

func TestWriteConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := buildDefaultConfig(FinishRequest{
		DBHost:     "localhost",
		DBPort:     5432,
		DBUser:     "postgres",
		DBPassword: "pass",
		DBName:     "erupe",
		Host:       "127.0.0.1",
		ClientMode: "ZZ",
	})

	if err := writeConfig(cfg); err != nil {
		t.Fatalf("writeConfig failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("reading config.json: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("config.json is not valid JSON: %v", err)
	}
	if parsed["Host"] != "127.0.0.1" {
		t.Errorf("Host = %v, want 127.0.0.1", parsed["Host"])
	}
}

func TestHandleIndex(t *testing.T) {
	ws := &wizardServer{
		logger: zap.NewNop(),
		done:   make(chan struct{}),
	}
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	ws.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	body := w.Body.String()
	if !contains(body, "Erupe Setup Wizard") {
		t.Error("response body missing wizard title")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
