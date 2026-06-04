"""
Pydantic-модели — Python-представление контракта данных
Зеркало Go-типов из internal/model/types.go.
"""
from __future__ import annotations
from datetime import datetime
from enum import Enum
from typing import Optional
from pydantic import BaseModel, Field


# Перечисления

class Status(str, Enum):
    """Результат одной CIS-проверки"""
    PASS  = "pass"
    FAIL  = "fail"
    WARN  = "warn"
    ERROR = "error"


class Severity(str, Enum):
    """Уровень опасности, если проверка провалена"""
    CRITICAL = "critical"
    HIGH     = "high"
    MEDIUM   = "medium"
    LOW      = "low"


# CIS-проверки (от Go-сервиса)

class CheckResult(BaseModel):
    """Результат одной CIS-проверки — зеркало Go-типа model.CheckResult"""
    id:            str
    title:         str
    category:      str                   # "image" или "container"
    cis_reference: Optional[str] = None  # None для best-practice проверок
    severity:      Severity
    status:        Status
    details:       str
    remediation:   str


class Target(BaseModel):
    """Что сканировали"""
    type: str   # "image" или "container"
    id:   str
    name: str


class Summary(BaseModel):
    """Сводка по результатам скана"""
    total:       int
    passed:      int
    failed:      int
    warned:      int
    score:       int  # passed / total * 100
    by_severity: dict[str, int] = Field(default_factory=dict)


class ScanReport(BaseModel):
    """Полный отчёт от Go-сервиса — зеркало model.ScanReport"""
    scanner_version: str
    scanned_at:      datetime
    target:          Target
    summary:         Summary
    checks:          list[CheckResult]


# Уязвимости (от Trivy)

class Vulnerability(BaseModel):
    """Одна CVE-уязвимость из вывода Trivy"""
    cve:               str
    package:           str
    severity:          str
    installed_version: str
    fixed_version:     Optional[str] = None
    description:       Optional[str] = None


class VulnSummary(BaseModel):
    """Количество уязвимостей по уровням серьёзности"""
    critical: int = 0
    high:     int = 0
    medium:   int = 0
    low:      int = 0
    unknown:  int = 0


# Объединённый отчёт оркестратора

class UnifiedReport(BaseModel):
    """
    Финальный отчёт: CIS-проверки (от Go) + CVE-уязвимости (от Trivy)
    Именно это возвращает FastAPI клиентам и дашборду
    """
    scan_id:         str
    scanned_at:      datetime
    target:          Target
    cis_report:      ScanReport           # 20 CIS-проверок
    vulnerabilities: list[Vulnerability]  # CVE от Trivy
    vuln_summary:    VulnSummary
    cis_summary:     Summary              # дублируем для удобства дашборда


# Модели запросов FastAPI

class ScanRequest(BaseModel):
    """Тело запроса POST /scan"""
    target_type: str    # "image" или "container"
    target:      str    # имя образа или ID контейнера
    with_cve:    bool = True  # нужно ли запускать Trivy


class ScanListItem(BaseModel):
    """Краткая запись в истории сканов (для GET /scans)"""
    scan_id:    str
    target:     Target
    scanned_at: datetime
    cis_score:  int
    cve_count:  int