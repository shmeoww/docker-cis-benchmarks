"""
Тесты для merger.py (merge_results) и report._safe_name.
Все unit-тесты — внешние сервисы не нужны.
"""
import pytest
from models import VulnSummary
from merger import merge_results
from report import _safe_name


# ─── merge_results ────────────────────────────────────────────────────────────

class TestMergeResults:

    def test_returns_unified_report(self, mock_scan_report, mock_vulnerabilities, mock_vuln_summary):
        """merge_results возвращает корректный UnifiedReport."""
        report = merge_results(mock_scan_report, mock_vulnerabilities, mock_vuln_summary)
        assert report.target == mock_scan_report.target
        assert report.cis_report == mock_scan_report
        assert report.vulnerabilities == mock_vulnerabilities
        assert report.cis_summary == mock_scan_report.summary

    def test_scan_id_is_unique(self, mock_scan_report, mock_vuln_summary):
        """Каждый вызов merge_results генерирует уникальный scan_id."""
        r1 = merge_results(mock_scan_report, [], mock_vuln_summary)
        r2 = merge_results(mock_scan_report, [], mock_vuln_summary)
        assert r1.scan_id != r2.scan_id

    def test_scan_id_is_uuid_format(self, mock_scan_report, mock_vuln_summary):
        """scan_id должен быть UUID-форматом."""
        import re
        report = merge_results(mock_scan_report, [], mock_vuln_summary)
        uuid_pattern = r'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
        assert re.match(uuid_pattern, report.scan_id), f"Неверный формат UUID: {report.scan_id}"

    def test_empty_vulnerabilities(self, mock_scan_report):
        """Работает с пустым списком уязвимостей."""
        report = merge_results(mock_scan_report, [], VulnSummary())
        assert report.vulnerabilities == []
        assert report.vuln_summary.critical == 0

    def test_scanned_at_is_utc(self, mock_scan_report, mock_vuln_summary):
        """scanned_at должен быть в UTC."""
        from datetime import timezone
        report = merge_results(mock_scan_report, [], mock_vuln_summary)
        assert report.scanned_at.tzinfo == timezone.utc

    def test_cis_summary_matches_report(self, mock_scan_report, mock_vuln_summary):
        """cis_summary дублирует summary из cis_report (для дашборда)."""
        report = merge_results(mock_scan_report, [], mock_vuln_summary)
        assert report.cis_summary.score == mock_scan_report.summary.score
        assert report.cis_summary.total == mock_scan_report.summary.total


# ─── _safe_name ───────────────────────────────────────────────────────────────

class TestSafeName:

    @pytest.mark.parametrize("input_name, expected", [
        ("mysql:8.0",             "mysql_8.0"),
        ("nginx:latest",          "nginx_latest"),
        ("/nats",                 "nats"),
        ("my_image",              "my_image"),
        ("gcr.io/project/image",  "gcr.io_project_image"),
        ("image:v1.2.3",          "image_v1.2.3"),
    ])
    def test_safe_name_parametrized(self, input_name, expected):
        assert _safe_name(input_name) == expected

    def test_empty_string_returns_unknown(self):
        assert _safe_name("") == "unknown"

    def test_only_slash_returns_unknown(self):
        assert _safe_name("/") == "unknown"