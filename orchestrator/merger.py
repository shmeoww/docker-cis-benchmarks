"""
Оркестрация и слияние результатов:
  Go-сервис: CIS-проверки (ScanReport)
  Trivy: CVE-уязвимости
  Merger: единый UnifiedReport
"""
import re
from datetime import datetime, timezone

from models import ScanReport, Vulnerability, VulnSummary, UnifiedReport
from scanner_client import ScannerClient
from trivy_client import scan_image_cve

def _make_scan_id(target_type: str, target_name: str) -> str:
    """
    Детерминированный scan_id на основе имени цели.
    'image' + 'mysql:8.0'  →  'image_mysql_8.0'
    Один и тот же образ получает один и тот же ID.
    """
    safe = target_name.lstrip("/")
    safe = re.sub(r"[:/\\]", "_", safe)
    safe = re.sub(r"[^\w\-.]", "", safe) or "unknown"
    return f"{target_type}_{safe}"

def merge_results(
    cis_report: ScanReport,
    vulnerabilities: list[Vulnerability],
    vuln_summary: VulnSummary,
    deterministic_id: bool = False, 
) -> UnifiedReport:
    """
    Чистая функция слияния: не вызывает внешние сервисы,
    просто собирает UnifiedReport из готовых данных
    """
    if deterministic_id:
        scan_id = _make_scan_id(cis_report.target.type, cis_report.target.name)
    else:
        import uuid
        scan_id = str(uuid.uuid4())
 
    return UnifiedReport(
        scan_id=scan_id,
        scanned_at=datetime.now(tz=timezone.utc),
        target=cis_report.target,
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
    cis_report = scanner.scan_image(image)
    vulns, vuln_summary = (scan_image_cve(image) if with_cve else ([], VulnSummary()))
    # deterministic_id=True — один образ = один ID в истории
    return merge_results(cis_report, vulns, vuln_summary, deterministic_id=True)


def orchestrate_container_scan(
    container_id: str,
    scanner: ScannerClient,
    with_cve: bool = True,
) -> UnifiedReport:
    cis_report = scanner.scan_container(container_id)
    if with_cve:
        vulns, vuln_summary = scan_image_cve(cis_report.target.name)
    else:
        vulns, vuln_summary = [], VulnSummary()
    return merge_results(cis_report, vulns, vuln_summary, deterministic_id=True)