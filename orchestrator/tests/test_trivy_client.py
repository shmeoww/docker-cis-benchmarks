"""
Тесты для trivy_client._parse_trivy_output.
Unit-тесты: подаём готовый JSON, проверяем парсинг. Trivy не нужен.
"""
import json
import pytest
from trivy_client import _parse_trivy_output


# ─── Вспомогательная функция ──────────────────────────────────────────────────

def make_trivy_json(vulnerabilities: list | None, target: str = "test") -> str:
    """Генерирует JSON в формате Trivy для тестов."""
    return json.dumps({
        "SchemaVersion": 2,
        "ArtifactName": target,
        "Results": [
            {
                "Target": target,
                "Class": "os-pkgs",
                "Type": "debian",
                "Vulnerabilities": vulnerabilities,
            }
        ]
    })


# ─── Тесты ────────────────────────────────────────────────────────────────────

class TestParseTrivyOutput:

    def test_parses_single_vulnerability(self):
        raw = make_trivy_json([{
            "VulnerabilityID": "CVE-2024-1234",
            "PkgName": "openssl",
            "InstalledVersion": "1.1.1n",
            "FixedVersion": "1.1.1w",
            "Severity": "HIGH",
            "Description": "Buffer overflow in openssl",
        }])
        vulns, summary = _parse_trivy_output(raw)
        assert len(vulns) == 1
        assert vulns[0].cve == "CVE-2024-1234"
        assert vulns[0].package == "openssl"
        assert vulns[0].severity == "high"   # lowercase
        assert vulns[0].installed_version == "1.1.1n"
        assert vulns[0].fixed_version == "1.1.1w"

    def test_summary_counts_by_severity(self):
        raw = make_trivy_json([
            {"VulnerabilityID": "CVE-1", "PkgName": "a", "InstalledVersion": "1",
             "Severity": "CRITICAL", "Description": ""},
            {"VulnerabilityID": "CVE-2", "PkgName": "b", "InstalledVersion": "1",
             "Severity": "HIGH", "Description": ""},
            {"VulnerabilityID": "CVE-3", "PkgName": "c", "InstalledVersion": "1",
             "Severity": "HIGH", "Description": ""},
            {"VulnerabilityID": "CVE-4", "PkgName": "d", "InstalledVersion": "1",
             "Severity": "MEDIUM", "Description": ""},
        ])
        vulns, summary = _parse_trivy_output(raw)
        assert summary.critical == 1
        assert summary.high == 2
        assert summary.medium == 1
        assert summary.low == 0
        assert len(vulns) == 4

    def test_null_vulnerabilities_returns_empty(self):
        """Trivy возвращает null вместо [] когда нет уязвимостей."""
        raw = make_trivy_json(None)
        vulns, summary = _parse_trivy_output(raw)
        assert vulns == []
        assert summary.critical == 0

    def test_empty_results_returns_empty(self):
        """Пустой массив Results."""
        raw = json.dumps({"Results": []})
        vulns, summary = _parse_trivy_output(raw)
        assert vulns == []

    def test_invalid_json_returns_empty(self):
        """Сломанный JSON не роняет сервис — возвращает пустой результат."""
        vulns, summary = _parse_trivy_output("это не json {{{{")
        assert vulns == []
        assert summary.critical == 0

    def test_multiple_results_sections(self):
        """Trivy может вернуть несколько секций (os + libs) — берём все."""
        raw = json.dumps({
            "Results": [
                {"Target": "os", "Vulnerabilities": [
                    {"VulnerabilityID": "CVE-OS", "PkgName": "libc",
                     "InstalledVersion": "1", "Severity": "LOW", "Description": ""},
                ]},
                {"Target": "libs", "Vulnerabilities": [
                    {"VulnerabilityID": "CVE-LIB", "PkgName": "requests",
                     "InstalledVersion": "2.0", "Severity": "MEDIUM", "Description": ""},
                ]},
            ]
        })
        vulns, summary = _parse_trivy_output(raw)
        assert len(vulns) == 2
        assert summary.low == 1
        assert summary.medium == 1

    def test_description_truncated_to_300_chars(self):
        """Длинные описания обрезаются до 300 символов."""
        long_desc = "A" * 500
        raw = make_trivy_json([{
            "VulnerabilityID": "CVE-LONG", "PkgName": "pkg",
            "InstalledVersion": "1", "Severity": "LOW",
            "Description": long_desc,
        }])
        vulns, _ = _parse_trivy_output(raw)
        assert len(vulns[0].description) <= 300

    def test_missing_fixed_version_is_none(self):
        """Если FixedVersion отсутствует — поле None."""
        raw = make_trivy_json([{
            "VulnerabilityID": "CVE-NO-FIX", "PkgName": "pkg",
            "InstalledVersion": "1", "Severity": "HIGH", "Description": "",
        }])
        vulns, _ = _parse_trivy_output(raw)
        assert vulns[0].fixed_version is None