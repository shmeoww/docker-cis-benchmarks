package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/moby/moby/client"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/checks"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/docker"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/model"
)

// testSetup создаёт роутер + клиент для тестов, пропускает если Docker недоступен.
func testSetup(t *testing.T) (*gin.Engine, *client.Client) {
	t.Helper()
	gin.SetMode(gin.TestMode) // убирает лишний вывод в тестах
	cli, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker недоступен: %v", err)
	}
	t.Cleanup(func() { cli.Close() })
	return setupRouter(cli), cli
}

// TestHealthEndpoint: GET /health → 200 + правильное тело
func TestHealthEndpoint(t *testing.T) {
	r, _ := testSetup(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("статус: got %d, want 200", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("status: got %q, want ok", body["status"])
	}
	if body["version"] != checks.Version {
		t.Errorf("version: got %q, want %q", body["version"], checks.Version)
	}
}

// TestScanImageBadRequest: POST без поля image → 400
func TestScanImageBadRequest(t *testing.T) {
	r, _ := testSetup(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scan/image",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("статус: got %d, want 400", w.Code)
	}
}

// TestScanImageEndpoint: POST /scan/image с реальным образом → 200 + ScanReport
func TestScanImageEndpoint(t *testing.T) {
	r, _ := testSetup(t)

	body, _ := json.Marshal(map[string]string{"image": "mysql:8.0"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scan/image", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("статус: got %d\nответ: %s", w.Code, w.Body.String())
	}
	var report model.ScanReport
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("ошибка декодирования: %v", err)
	}
	if report.Target.Type != "image" {
		t.Errorf("Target.Type: got %q, want image", report.Target.Type)
	}
	if len(report.Checks) == 0 {
		t.Error("Checks не должны быть пустыми")
	}
}

// TestScanContainerBadRequest: POST без поля container → 400
func TestScanContainerBadRequest(t *testing.T) {
	r, _ := testSetup(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scan/container",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("статус: got %d, want 400", w.Code)
	}
}

// TestScanContainerEndpoint: POST /scan/container с реальным контейнером → 200
func TestScanContainerEndpoint(t *testing.T) {
	r, cli := testSetup(t)

	containers, err := cli.ContainerList(context.Background(),
		client.ContainerListOptions{All: true})
	if err != nil || len(containers.Items) == 0 {
		t.Skip("нет контейнеров для тестирования")
	}
	containerID := containers.Items[0].ID

	body, _ := json.Marshal(map[string]string{"container": containerID})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scan/container", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("статус: got %d\nответ: %s", w.Code, w.Body.String())
	}
	var report model.ScanReport
	json.NewDecoder(w.Body).Decode(&report)
	if report.Target.Type != "container" {
		t.Errorf("Target.Type: got %q, want container", report.Target.Type)
	}
}

// TestScanAllEndpoint: POST /scan/all → 200 + поле scans
func TestScanAllEndpoint(t *testing.T) {
	r, _ := testSetup(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scan/all", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("статус: got %d", w.Code)
	}
	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if _, ok := result["scans"]; !ok {
		t.Error("ответ должен содержать поле scans")
	}
}