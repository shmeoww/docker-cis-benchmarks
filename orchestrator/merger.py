"""
Оркестрация и слияние результатов:
  Go-сервис: CIS-проверки (ScanReport)
  Trivy: CVE-уязвимости
  Merger: единый UnifiedReport
"""
import uuid
from datetime import datetime, timezone

from models import ScanReport, Vulnerability, VulnSummary, UnifiedReport
from scanner_client import ScannerClient
from trivy_client import scan_image_cve


def merge_results(
    cis_report: ScanReport,
    vulnerabilities: list[Vulnerability],
    vuln_summary: VulnSummary,
) -> UnifiedReport:
    """
    Чистая функция слияния: не вызывает внешние сервисы,
    просто собирает UnifiedReport из готовых данных
    """
    return UnifiedReport(
        scan_id=str(uuid.uuid4()),          # уникальный ID скана
        scanned_at=datetime.now(tz=timezone.utc),
        target=cis_report.target,           # берём цель из CIS-отчёта
        cis_report=cis_report,
        vulnerabilities=vulnerabilities,
        vuln_summary=vuln_summary,
        cis_summary=cis_report.summary,
    )


def orchestrate_image_scan(
    image: str,
    scanner: ScannerClient,
    with_cve: bool = True,
) -> UnifiedReport:
    """
    Полный скан образа: CIS-проверки + (опционально) CVE, вызывается из FastAPI-эндпоинта
    """
    # 1. CIS-проверки через Go-сервис
    cis_report = scanner.scan_image(image)

    # 2. CVE-скан через Trivy (если запрошен)
    if with_cve:
        vulns, vuln_summary = scan_image_cve(image)
    else:
        vulns, vuln_summary = [], VulnSummary()

    # 3. Объединяем и возвращаем
    return merge_results(cis_report, vulns, vuln_summary)


def orchestrate_container_scan(
    container_id: str,
    scanner: ScannerClient,
    with_cve: bool = True,
) -> UnifiedReport:
    """
    Полный скан контейнера: CIS-проверки + (опционально) CVE по образу
    Для CVE используем образ контейнера — именно он содержит пакеты
    """
    cis_report = scanner.scan_container(container_id)

    if with_cve:
        # Сканируем образ контейнера, а не контейнер напрямую
        image_name = cis_report.target.name
        vulns, vuln_summary = scan_image_cve(image_name)
    else:
        vulns, vuln_summary = [], VulnSummary()

    return merge_results(cis_report, vulns, vuln_summary)