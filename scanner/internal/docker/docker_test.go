package docker

import (
	"context"
	"testing"

	"github.com/moby/moby/client"
)

// TestCollectImage — интеграционный тест: реальный вызов Docker API.
// Пропускается при go test -short
func TestCollectImage(t *testing.T) {
	if testing.Short() {
		t.Skip("интеграционный тест — требует запущенный Docker")
	}

	cli, err := NewClient()
	if err != nil {
		t.Fatalf("не удалось создать Docker-клиент: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()

	// Используем образ, который точно есть в её системе
	data, err := CollectImage(ctx, cli, "mirror.gcr.io/library/nats:2.10-alpine")
	if err != nil {
		t.Skipf("образ недоступен, пропускаем: %v", err)
	}

	if data.ID == "" {
		t.Error("ID образа не должен быть пустым")
	}
	if len(data.Tags) == 0 {
		t.Error("Tags не должны быть пустыми для именованного образа")
	}
	// nats — не запускается от root по умолчанию
	// Просто проверяем, что поле заполнено (не паникует)
	t.Logf("User: %q, HasHealthcheck: %v, Ports: %v",
		data.User, data.HasHealthcheck, data.ExposedPorts)
}

// TestCollectContainer — интеграционный тест для сбора данных контейнера.
func TestCollectContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("интеграционный тест — требует запущенный Docker")
	}

	cli, err := NewClient()
	if err != nil {
		t.Fatalf("не удалось создать Docker-клиент: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()

	// Берём первый доступный контейнер
	containers, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil || len(containers.Items) == 0 {
		t.Skip("нет контейнеров для тестирования")
	}

	data, err := CollectContainer(ctx, cli, containers.Items[0].ID)
	if err != nil {
		t.Fatalf("CollectContainer вернул ошибку: %v", err)
	}

	if data.ID == "" {
		t.Error("ID контейнера не должен быть пустым")
	}
	if data.Name == "" {
		t.Error("Name контейнера не должно быть пустым")
	}
	t.Logf("Контейнер: %s, Privileged: %v, NetworkMode: %s",
		data.Name, data.Privileged, data.NetworkMode)
}