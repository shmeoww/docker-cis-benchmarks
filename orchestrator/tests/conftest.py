"""
Общие pytest-fixture'ы для всех тестов оркестратора.
Создают готовые объекты моковых данных — не нужно повторяться в каждом тесте.
"""
from datetime import datetime, timezone

import pytest
from fastapi.testclient import TestClient

from models import (
    CheckResult, Severity, Status,
    ScanReport, Target, Summary,
    Vulnerability, VulnSummary, UnifiedReport,
)


# Моковые CIS-проверки

@pytest.fixture
def mock_checks():
    """Два результата CIS-проверки (fail + pass) для тестов."""
    return [
        CheckResult(
            id="image_no_root_user", title="Не root",
            category="image", cis_reference="4.1",
            severity=Severity.CRITICAL, status=Status.FAIL,
            details="USER не задан", remediation="Добавьте USER",
        ),
        CheckResult(
            id="image_no_latest_tag", title="Не latest",
            category="image",
            severity=Severity.MEDIUM, status=Status.PASS,
            details="Версия зафиксирована: mysql:8.0", remediation="",
        ),
    ]


@pytest.fixture
def mock_target():
    return Target(type="image", id="sha256:abc123", name="mysql:8.0")


@pytest.fixture
def mock_summary():
    return Summary(
        total=2, passed=1, failed=1, warned=0,
        score=50, by_severity={"critical": 1},
    )


@pytest.fixture
def mock_scan_report(mock_target, mock_summary, mock_checks):
    """Полный ScanReport от Go-сервиса."""
    return ScanReport(
        scanner_version="0.1.0",
        scanned_at=datetime(2026, 6, 1, 12, 0, tzinfo=timezone.utc),
        target=mock_target,
        summary=mock_summary,
        checks=mock_checks,
    )


@pytest.fixture
def mock_vulnerabilities():
    """Небольшой список CVE-уязвимостей."""
    return [
        Vulnerability(
            cve="CVE-2024-1234", package="openssl",
            severity="high", installed_version="1.1.1n",
            fixed_version="1.1.1w",
        ),
        Vulnerability(
            cve="CVE-2024-5678", package="libc",
            severity="critical", installed_version="2.31",
            fixed_version=None,
        ),
    ]


@pytest.fixture
def mock_vuln_summary():
    return VulnSummary(critical=1, high=1, medium=0, low=0)


@pytest.fixture
def mock_unified_report(mock_scan_report, mock_vulnerabilities, mock_vuln_summary):
    """Полный UnifiedReport — главный объект оркестратора."""
    return UnifiedReport(
        scan_id="test-scan-id-0001",
        scanned_at=datetime(2026, 6, 1, 12, 0, tzinfo=timezone.utc),
        target=mock_scan_report.target,
        cis_report=mock_scan_report,
        vulnerabilities=mock_vulnerabilities,
        vuln_summary=mock_vuln_summary,
        cis_summary=mock_scan_report.summary,
    )


# FastAPI TestClient

@pytest.fixture
def client():
    """
    TestClient для FastAPI — не запускает реальный сервер,
    работает прямо в памяти. Не требует запущенного Go-сервиса
    (эндпоинты с ошибками тестируем без него).
    """
    from main import app
    return TestClient(app, raise_server_exceptions=False)


# Маркер для интеграционных тестов

@pytest.fixture
def require_go_service():
    """
    Пропускает тест если Go-сервис недоступен.
    Используй как параметр функции: def test_something(require_go_service, ...):
    """
    from scanner_client import ScannerClient
    if not ScannerClient().health():
        pytest.skip("Go-сервис недоступен (запусти: cd scanner && go run .)")