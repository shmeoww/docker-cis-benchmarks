package checks

import (
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/docker"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/model"
)

// ImageCheck описывает одну CIS-проверку для Docker-образа.
// Метаданные задаются один раз в структуре; логика — в поле Eval.
type ImageCheck struct {
	ID           string
	Title        string
	CISReference string         // номер пункта CIS, напр. "4.1"; пусто для best-practice
	Severity     model.Severity
	// Eval получает данные образа и возвращает: статус, доказательство, рекомендацию.
	Eval func(data docker.ImageData) (status model.Status, details string, remediation string)
}

// Run запускает проверку и возвращает полный CheckResult.
// Метаданные (ID, Title и т.д.) берутся из структуры — не нужно повторять их в каждой Eval-функции.
func (c ImageCheck) Run(data docker.ImageData) model.CheckResult {
	status, details, remediation := c.Eval(data)
	return model.CheckResult{
		ID:           c.ID,
		Title:        c.Title,
		Category:     "image",
		CISReference: c.CISReference,
		Severity:     c.Severity,
		Status:       status,
		Details:      details,
		Remediation:  remediation,
	}
}

// ContainerCheck описывает одну CIS-проверку для запущенного контейнера.
type ContainerCheck struct {
	ID           string
	Title        string
	CISReference string
	Severity     model.Severity
	Eval         func(data docker.ContainerData) (status model.Status, details string, remediation string)
}

// Run запускает проверку на данных контейнера и возвращает полный CheckResult.
func (c ContainerCheck) Run(data docker.ContainerData) model.CheckResult {
	status, details, remediation := c.Eval(data)
	return model.CheckResult{
		ID:           c.ID,
		Title:        c.Title,
		Category:     "container",
		CISReference: c.CISReference,
		Severity:     c.Severity,
		Status:       status,
		Details:      details,
		Remediation:  remediation,
	}
}