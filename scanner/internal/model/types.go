package model

import "time"

type Status string

const (
	StatusPass  Status = "pass"  // проверка пройдена
	StatusFail  Status = "fail"  // проверка провалена
	StatusWarn  Status = "warn"  // замечание
	StatusError Status = "error" // не удалось выполнить проверку
)

// Severity — уровень опасности, если проверка провалена.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// CheckResult — результат одной CIS-проверки
// JSON-теги задают имена полей в выходном JSON.
type CheckResult struct {
	ID           string   `json:"id"`                       
	Title        string   `json:"title"`                    
	Category     string   `json:"category"`                 // "image" или "container"
	CISReference string   `json:"cis_reference,omitempty"`  
	Severity     Severity `json:"severity"`
	Status       Status   `json:"status"`
	Details      string   `json:"details"`      
	Remediation  string   `json:"remediation"`  // как исправить
}

// Target — описание того, что сканировали.
type Target struct {
	Type string `json:"type"` // "image" или "container"
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Summary — сводка по результатам скана.
type Summary struct {
	Total      int            `json:"total"`
	Passed     int            `json:"passed"`
	Failed     int            `json:"failed"`
	Warned     int            `json:"warned"`
	Score      int            `json:"score"`       // passed / total * 100
	BySeverity map[string]int `json:"by_severity"` // число провалов по уровням опасности
}

// ScanReport — полный отчёт об одном скане
type ScanReport struct {
	ScannerVersion string        `json:"scanner_version"`
	ScannedAt      time.Time     `json:"scanned_at"`
	Target         Target        `json:"target"`
	Summary        Summary       `json:"summary"`
	Checks         []CheckResult `json:"checks"`
}

// ComputeSummary подсчитывает сводку по списку результатов и возвращает заполненный Summary
func ComputeSummary(checks []CheckResult) Summary {
	s := Summary{
		Total:      len(checks),
		BySeverity: make(map[string]int),
	}
	for _, c := range checks {
		switch c.Status {
		case StatusPass:
			s.Passed++
		case StatusFail:
			s.Failed++
			s.BySeverity[string(c.Severity)]++ // считаем провалы по уровню
		case StatusWarn:
			s.Warned++
		}
	}
	if s.Total > 0 {
		s.Score = s.Passed * 100 / s.Total
	}
	return s
}