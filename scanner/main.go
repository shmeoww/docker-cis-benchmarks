package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/model"
)

func main() {
	// Проверяем, что типы сериализуются в JSON точно по контракту из п. 1.1.
	check := model.CheckResult{
		ID:           "container_no_privileged",
		Title:        "Контейнер не должен запускаться в привилегированном режиме",
		Category:     "container",
		CISReference: "5.4",
		Severity:     model.SeverityCritical,
		Status:       model.StatusFail,
		Details:      "HostConfig.Privileged = true",
		Remediation:  "Перезапустите контейнер без флага --privileged.",
	}

	b, err := json.MarshalIndent(check, "", "  ")
	if err != nil {
		log.Fatalf("ошибка сериализации: %v", err)
	}
	fmt.Println("Один CheckResult в JSON:")
	fmt.Println(string(b))

	// Проверяем ComputeSummary
	checks := []model.CheckResult{
		{Status: model.StatusPass, Severity: model.SeverityHigh},
		{Status: model.StatusFail, Severity: model.SeverityCritical},
		{Status: model.StatusFail, Severity: model.SeverityHigh},
		{Status: model.StatusWarn, Severity: model.SeverityMedium},
	}
	summary := model.ComputeSummary(checks)
	sb, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println("\nSummary для 4 проверок:")
	fmt.Println(string(sb))
}