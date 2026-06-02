package main

import (
	"context"
	"fmt"
	"log"

	"github.com/moby/moby/client"
)

func main() {
	// context — механизм отмены и таймаутов для запросов. Пока берём пустой.
	ctx := context.Background()

	// Создаём клиент Docker через новую библиотеку moby.
	// client.FromEnv берёт настройки подключения из окружения
	// (на Windows — автоматически именованный канал Docker Desktop).
	// Согласование версии API здесь включено по умолчанию.
	cli, err := client.New(client.FromEnv)
	if err != nil {
		log.Fatalf("не удалось создать Docker-клиент: %v", err)
	}
	defer cli.Close() // закрыть соединение при выходе из main

	// --- Проверка 1: список образов ---
	images, err := cli.ImageList(ctx, client.ImageListOptions{})
	if err != nil {
		log.Fatalf("не удалось получить список образов: %v", err)
	}
	fmt.Printf("Найдено образов: %d\n", len(images.Items))
	for _, img := range images.Items {
		name := "<без тега>"
		if len(img.RepoTags) > 0 {
			name = img.RepoTags[0]
		}
		fmt.Printf("  - %s\n", name)
	}

	// --- Проверка 2: список контейнеров (все, включая остановленные) ---
	containers, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("не удалось получить список контейнеров: %v", err)
	}
	fmt.Printf("\nНайдено контейнеров: %d\n", len(containers.Items))
	for _, c := range containers.Items {
		name := "<без имени>"
		if len(c.Names) > 0 {
			name = c.Names[0]
		}
		fmt.Printf("  - %s (образ: %s) — %s\n", name, c.Image, c.Status)
	}
}