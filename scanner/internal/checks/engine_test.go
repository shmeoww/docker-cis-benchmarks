package checks

import (
	"context"
	"testing"

	"github.com/moby/moby/client"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/docker"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/model"
)

func TestScanImage(t *testing.T) {
	if testing.Short() {
		t.Skip("интеграционный тест — требует Docker")
	}
	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("Docker-клиент: %v", err)
	}
	defer cli.Close()

	report, err := ScanImage(context.Background(), cli, "mysql:8.0")
	if err != nil {
		t.Fatalf("ScanImage: %v", err)
	}

	if report.Target.Type != "image" {
		t.Errorf("Target.Type: got %q, want image", report.Target.Type)
	}
	if report.Target.Name != "mysql:8.0" {
		t.Errorf("Target.Name: got %q, want mysql:8.0", report.Target.Name)
	}
	if report.ScannerVersion != Version {
		t.Errorf("ScannerVersion: got %q, want %q", report.ScannerVersion, Version)
	}
	if len(report.Checks) != len(ImageChecks) {
		t.Errorf("Checks: got %d, want %d", len(report.Checks), len(ImageChecks))
	}
	if report.Summary.Total != len(ImageChecks) {
		t.Errorf("Summary.Total: got %d, want %d", report.Summary.Total, len(ImageChecks))
	}
	// mysql:8.0 запускается от root — ожидаем fail для этой проверки
	for _, c := range report.Checks {
		if c.ID == "image_no_root_user" && c.Status != model.StatusFail {
			t.Errorf("mysql:8.0 должен провалить image_no_root_user, got %s", c.Status)
		}
	}
	t.Logf("mysql:8.0 оценка: %d/100, провалено: %d", report.Summary.Score, report.Summary.Failed)
}

func TestScanContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("интеграционный тест — требует Docker")
	}
	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("Docker-клиент: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil || len(containers.Items) == 0 {
		t.Skip("нет контейнеров")
	}

	report, err := ScanContainer(ctx, cli, containers.Items[0].ID)
	if err != nil {
		t.Fatalf("ScanContainer: %v", err)
	}

	if report.Target.Type != "container" {
		t.Errorf("Target.Type: got %q, want container", report.Target.Type)
	}
	if len(report.Checks) != len(ContainerChecks) {
		t.Errorf("Checks: got %d, want %d", len(report.Checks), len(ContainerChecks))
	}
	if report.Summary.Total != len(ContainerChecks) {
		t.Errorf("Summary.Total: got %d, want %d", report.Summary.Total, len(ContainerChecks))
	}
	t.Logf("%s оценка: %d/100", report.Target.Name, report.Summary.Score)
}

func TestScanAll(t *testing.T) {
	if testing.Short() {
		t.Skip("интеграционный тест — требует Docker")
	}
	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("Docker-клиент: %v", err)
	}
	defer cli.Close()

	reports, err := ScanAll(context.Background(), cli)
	if err != nil {
		t.Fatalf("ScanAll: %v", err)
	}
	if len(reports) == 0 {
		t.Error("ScanAll должен вернуть хотя бы один отчёт")
	}
	for _, r := range reports {
		if r.Target.ID == "" {
			t.Errorf("отчёт без Target.ID: %s", r.Target.Name)
		}
	}
	t.Logf("Всего просканировано: %d целей", len(reports))
}

// Тесты error-путей в engine.go (строчки: return model.ScanReport{}, err)
 
func TestScanImageError(t *testing.T) {
	if testing.Short() {
		t.Skip("требует Docker")
	}
	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("Docker-клиент: %v", err)
	}
	defer cli.Close()
 
	// Несуществующий образ → Docker вернёт ошибку → ScanImage должна её вернуть
	_, err = ScanImage(context.Background(), cli, "nonexistent_image_xyz:v999999")
	if err == nil {
		t.Error("ожидалась ошибка для несуществующего образа, получили nil")
	}
}
 
func TestScanContainerError(t *testing.T) {
	if testing.Short() {
		t.Skip("требует Docker")
	}
	cli, err := docker.NewClient()
	if err != nil {
		t.Fatalf("Docker-клиент: %v", err)
	}
	defer cli.Close()
 
	// Несуществующий контейнер → Docker вернёт ошибку
	_, err = ScanContainer(context.Background(), cli, "nonexistent_container_xyz_99999")
	if err == nil {
		t.Error("ожидалась ошибка для несуществующего контейнера, получили nil")
	}
}