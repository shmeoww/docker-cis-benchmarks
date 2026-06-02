package main

import (
	"context"
	"fmt"
	"log"

	"github.com/moby/moby/client"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/docker"
)

func main() {
	ctx := context.Background()

	// Клиент теперь создаём через наш собственный пакет docker.
	cli, err := docker.NewClient()
	if err != nil {
		log.Fatalf("не удалось создать Docker-клиент: %v", err)
	}
	defer cli.Close()

	// --- список образов ---
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

	// --- список контейнеров ---
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