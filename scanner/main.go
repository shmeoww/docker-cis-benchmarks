package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/moby/moby/client"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/checks"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/docker"
)

func main() {
	ctx := context.Background()

	cli, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Docker-клиент: %v", err)
	}
	defer cli.Close()

	// --- Полный отчёт по образу через engine ---
	fmt.Println("Сканируем образ mysql:8.0 ...")
	imageReport, err := checks.ScanImage(ctx, cli, "mysql:8.0")
	if err != nil {
		log.Fatalf("ScanImage: %v", err)
	}
	printReport(imageReport)

	// --- Полный отчёт по первому контейнеру через engine ---
	containers, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("ContainerList: %v", err)
	}
	if len(containers.Items) > 0 {
		fmt.Printf("Сканируем контейнер %s ...\n", containers.Items[0].Names[0])
		containerReport, err := checks.ScanContainer(ctx, cli, containers.Items[0].ID)
		if err != nil {
			log.Fatalf("ScanContainer: %v", err)
		}
		printReport(containerReport)
	}
}

func printReport(report interface{}) {
	b, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(b))
	fmt.Println()
}