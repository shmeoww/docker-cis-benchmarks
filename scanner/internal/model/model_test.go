package model

import "testing"

func TestComputeSummary(t *testing.T) {
	checks := []CheckResult{
		{Status: StatusPass, Severity: SeverityHigh},
		{Status: StatusFail, Severity: SeverityCritical},
		{Status: StatusFail, Severity: SeverityHigh},
		{Status: StatusWarn, Severity: SeverityMedium},
		{Status: StatusError, Severity: SeverityLow},
	}
	s := ComputeSummary(checks)

	tests := []struct {
		name string
		got  int
		want int
	}{
		{"Total", s.Total, 5},
		{"Passed", s.Passed, 1},
		{"Failed", s.Failed, 2},
		{"Warned", s.Warned, 1},
		{"Score", s.Score, 20}, // 1 пройдено из 5 → 20%
		{"BySeverity critical", s.BySeverity["critical"], 1},
		{"BySeverity high", s.BySeverity["high"], 1},
		{"BySeverity medium", s.BySeverity["medium"], 0}, // warn не считается
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %d, want %d", tt.got, tt.want)
			}
		})
	}
}

func TestComputeSummaryEmpty(t *testing.T) {
	s := ComputeSummary(nil)
	if s.Total != 0 {
		t.Errorf("Total: got %d, want 0", s.Total)
	}
	if s.Score != 0 {
		t.Errorf("Score: got %d, want 0 (деление на ноль)", s.Score)
	}
}

func TestComputeSummaryAllPassed(t *testing.T) {
	checks := []CheckResult{
		{Status: StatusPass, Severity: SeverityHigh},
		{Status: StatusPass, Severity: SeverityMedium},
	}
	s := ComputeSummary(checks)
	if s.Score != 100 {
		t.Errorf("Score: got %d, want 100 (все прошли)", s.Score)
	}
	if len(s.BySeverity) != 0 {
		t.Errorf("BySeverity должна быть пустой (нет провалов), got %v", s.BySeverity)
	}
}