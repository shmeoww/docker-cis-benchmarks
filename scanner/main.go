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

	cli, err := docker.NewClient()
	if err != nil {
		log.Fatalf("не удалось создать Docker-клиент: %v", err)
	}
	defer cli.Close()

	// Берём первый контейнер из списка — для проверки сборщика.
	containers, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("не удалось получить список контейнеров: %v", err)
	}
	if len(containers.Items) == 0 {
		log.Fatal("нет контейнеров для проверки")
	}
	ref := containers.Items[0].ID

	data, err := docker.CollectContainer(ctx, cli, ref)
	if err != nil {
		log.Fatalf("не удалось собрать данные по контейнеру: %v", err)
	}

	fmt.Printf("Контейнер: %s\n", data.Name)
	fmt.Printf("  Образ:             %s\n", data.Image)
	fmt.Printf("  Privileged:        %v\n", data.Privileged)
	fmt.Printf("  Доб. capabilities: %v\n", data.AddedCaps)
	fmt.Printf("  NetworkMode:       %q\n", data.NetworkMode)
	fmt.Printf("  PidMode:           %q\n", data.PidMode)
	fmt.Printf("  IpcMode:           %q\n", data.IpcMode)
	fmt.Printf("  Монтирования:      %v\n", data.Binds)
	fmt.Printf("  SecurityOpt:       %v\n", data.SecurityOpt)
	fmt.Printf("  ReadonlyRootfs:    %v\n", data.ReadonlyRootfs)
	fmt.Printf("  Лимит памяти:      %d\n", data.MemoryLimit)
	fmt.Printf("  Лимит CPU (nano):  %d\n", data.NanoCPUs)
	fmt.Printf("  Лимит PIDs:        %d\n", data.PidsLimit)
	fmt.Printf("  Restart policy:    %q (max retry %d)\n", data.RestartPolicy, data.MaxRetry)
}