"""
Клиент к Trivy: запускает trivy image --format json,
парсит вывод и возвращает список уязвимостей и сводку.
"""
import json
import subprocess
from models import Vulnerability, VulnSummary


def scan_image_cve(
    image: str,
    timeout: int = 300,
) -> tuple[list[Vulnerability], VulnSummary]:
    """
    Запускает Trivy для поиска CVE-уязвимостей в образе.

    Возвращает (vulnerabilities, summary).
    Если Trivy недоступен или образ не найден — возвращает пустые результаты
    (не бросает исключение, чтобы не ломать остальной скан).
    """
    try:
        result = subprocess.run(
            [
                "trivy", "image",
                "--format", "json",   # вывод в JSON
                "--quiet",            # прогресс-бары в stderr, JSON в stdout
                "--scanners", "vuln", # только уязвимости (не секреты)
                image,
            ],
            capture_output=True,
            text=True,
            timeout=timeout,
        )
    except FileNotFoundError:
        # trivy не установлен
        print("[trivy] предупреждение: trivy не найден, CVE-скан пропущен")
        return [], VulnSummary()
    except subprocess.TimeoutExpired:
        print(f"[trivy] таймаут при сканировании {image}")
        return [], VulnSummary()

    if result.returncode != 0:
        print(f"[trivy] ошибка (code={result.returncode}): {result.stderr[:200]}")
        return [], VulnSummary()

    return _parse_trivy_output(result.stdout)


def _parse_trivy_output(
    raw_json: str,
) -> tuple[list[Vulnerability], VulnSummary]:
    """
    Разбирает JSON-вывод Trivy и возвращает плоский список уязвимостей
    и сводку по уровням серьёзности.

    Структура Trivy JSON:
    {
      "Results": [
        {
          "Vulnerabilities": [
            { "VulnerabilityID": "CVE-...", "PkgName": "...",
              "InstalledVersion": "...", "FixedVersion": "...",
              "Severity": "HIGH", ... }
          ]
        }
      ]
    }
    """
    try:
        data = json.loads(raw_json)
    except json.JSONDecodeError:
        print("[trivy] не удалось разобрать JSON-вывод")
        return [], VulnSummary()

    vulns: list[Vulnerability] = []
    summary = VulnSummary()

    for result in data.get("Results", []):
        for vuln in result.get("Vulnerabilities") or []:
            severity = vuln.get("Severity", "UNKNOWN").lower()

            v = Vulnerability(
                cve=vuln.get("VulnerabilityID", "UNKNOWN"),
                package=vuln.get("PkgName", ""),
                severity=severity,
                installed_version=vuln.get("InstalledVersion", ""),
                fixed_version=vuln.get("FixedVersion"),
                description=vuln.get("Description", "")[:300] or None,
            )
            vulns.append(v)

            # Обновляем счётчик сводки
            if severity == "critical":
                summary.critical += 1
            elif severity == "high":
                summary.high += 1
            elif severity == "medium":
                summary.medium += 1
            elif severity == "low":
                summary.low += 1
            else:
                summary.unknown += 1

    return vulns, summary