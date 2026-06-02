package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/checks"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/docker"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/model"
)

func main() {
	ctx := context.Background()

	cli, err := docker.NewClient()
	if err != nil {
		log.Fatalf("не удалось создать Docker-клиент: %v", err)
	}
	defer cli.Close()

	// Образ для тестирования проверок.
	ref := "mysql:8.0"
	fmt.Printf("=== Сканируем образ: %s ===\n\n", ref)

	data, err := docker.CollectImage(ctx, cli, ref)
	if err != nil {
		log.Fatalf("не удалось собрать данные образа: %v", err)
	}

	// Прогоняем все проверки образов.
	var results []model.CheckResult
	for _, check := range checks.ImageChecks {
		results = append(results, check.Run(data))
	}

	// Выводим каждый результат в JSON.
	for _, r := range results {
		b, _ := json.MarshalIndent(r, "", "  ")
		fmt.Println(string(b))
		fmt.Println()
	}

	// Итоговая сводка.
	summary := model.ComputeSummary(results)
	fmt.Printf("Оценка: %d/100 | Пройдено: %d | Провалено: %d | Предупреждений: %d\n",
		summary.Score, summary.Passed, summary.Failed, summary.Warned)
}