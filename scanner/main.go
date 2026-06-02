package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/moby/moby/client"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/checks"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/docker"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/model"
)

func printResults(label string, results []model.CheckResult) {
	fmt.Printf("=== %s ===\n\n", label)
	for _, r := range results {
		b, _ := json.MarshalIndent(r, "", "  ")
		fmt.Println(string(b))
		fmt.Println()
	}
	s := model.ComputeSummary(results)
	fmt.Printf("Оценка: %d/100 | ✓ %d | ✗ %d | ⚠ %d\n\n",
		s.Score, s.Passed, s.Failed, s.Warned)
}

func main() {
	ctx := context.Background()

	cli, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Docker-клиент: %v", err)
	}
	defer cli.Close()

	// --- Проверяем образ ---
	imageData, err := docker.CollectImage(ctx, cli, "mysql:8.0")
	if err != nil {
		log.Fatalf("CollectImage: %v", err)
	}
	var imageResults []model.CheckResult
	for _, c := range checks.ImageChecks {
		imageResults = append(imageResults, c.Run(imageData))
	}
	printResults("Образ mysql:8.0 — 6 проверок", imageResults)

	// --- Проверяем первый контейнер из списка ---
	containers, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("ContainerList: %v", err)
	}
	if len(containers.Items) == 0 {
		log.Fatal("нет контейнеров")
	}
	ref := containers.Items[0].ID
	name := containers.Items[0].Names[0]

	containerData, err := docker.CollectContainer(ctx, cli, ref)
	if err != nil {
		log.Fatalf("CollectContainer: %v", err)
	}
	var containerResults []model.CheckResult
	for _, c := range checks.ContainerChecks {
		containerResults = append(containerResults, c.Run(containerData))
	}
	printResults("Контейнер "+name+" — 14 проверок", containerResults)
}