"""
FastAPI-оркестратор: принимает задания на скан, координирует
Go-сервис (CIS-проверки) и Trivy (CVE), возвращает UnifiedReport
Запуск: uvicorn main:app --reload --port 8080
Документация: http://localhost:8080/docs
"""
from __future__ import annotations

import requests as req_lib
from fastapi import FastAPI, HTTPException
from fastapi.responses import FileResponse

from models import UnifiedReport, ScanRequest, ScanListItem
from scanner_client import ScannerClient
from merger import orchestrate_image_scan, orchestrate_container_scan
from report import save_html_report

# FastAPI-приложение
app = FastAPI(
    title="Docker CIS Scanner",
    version="0.1.0",
    description=(
        "Оркестратор безопасности Docker-образов и контейнеров. "
        "Объединяет CIS Benchmark-проверки (Go-сервис) "
        "и CVE-уязвимости (Trivy)."
    ),
)

# Глобальные объекты
# Клиент к Go-сервису — создаётся один раз при старте.
# URL берётся из переменной окружения SCANNER_URL (по умолчанию localhost:8000).
scanner = ScannerClient()

# История сканов в памяти: scan_id - UnifiedReport. Сбрасывается при перезапуске сервера
_results: dict[str, UnifiedReport] = {}


# Служебные эндпоинты

@app.get("/health", tags=["Служебные"])
def health():
    """Проверка работоспособности оркестратора и Go-сервиса."""
    return {
        "status": "ok",
        "go_scanner": "online" if scanner.health() else "offline",
    }


# Сканирование

@app.post("/scan", response_model=UnifiedReport, tags=["Сканирование"])
def scan(request: ScanRequest):
    """
    Запустить полный скан цели.

    - **target_type**: "image" или "container"
    - **target**: имя образа ("mysql:8.0") или имя/ID контейнера
    - **with_cve**: запускать CVE-скан через Trivy (по умолчанию "true")

    Возвращает UnifiedReport: CIS-проверки + CVE-уязвимости
    Отчёт сохраняется в "reports/" и доступен через "GET /report/{scan_id}".
    """
    try:
        if request.target_type == "image":
            report = orchestrate_image_scan(
                request.target, scanner, request.with_cve
            )
        elif request.target_type == "container":
            report = orchestrate_container_scan(
                request.target, scanner, request.with_cve
            )
        else:
            raise HTTPException(
                status_code=400,
                detail="target_type должен быть 'image' или 'container'",
            )
    except HTTPException:
        raise  # HTTPException (400, 404 и т.д.)
    except req_lib.ConnectionError:
        raise HTTPException(
            status_code=503,
            detail="Go-сервис недоступен. Убедитесь что scanner запущен на порту 8000.",
        )
    except Exception as exc:
        raise HTTPException(status_code=500, detail=str(exc))

    _results[report.scan_id] = report
    save_html_report(report)
    return report


# История сканов

@app.get("/scans", response_model=list[ScanListItem], tags=["История"])
def list_scans():
    """Список всех выполненных сканов (краткая информация)"""
    return [
        ScanListItem(
            scan_id=r.scan_id,
            target=r.target,
            scanned_at=r.scanned_at,
            cis_score=r.cis_summary.score,
            cve_count=len(r.vulnerabilities),
        )
        for r in _results.values()
    ]


@app.get("/scans/{scan_id}", response_model=UnifiedReport, tags=["История"])
def get_scan(scan_id: str):
    """Полный UnifiedReport по ID скана"""
    report = _results.get(scan_id)
    if not report:
        raise HTTPException(
            status_code=404, detail=f"Скан {scan_id!r} не найден"
        )
    return report


# HTML-отчёты

@app.get("/report/{scan_id}", tags=["Отчёты"])
def get_html_report(scan_id: str):
    """HTML-отчёт по ID скана (открывается напрямую в браузере)."""
    report = _results.get(scan_id)
    if not report:
        raise HTTPException(
            status_code=404, detail=f"Скан {scan_id!r} не найден"
        )
    path = save_html_report(report)
    return FileResponse(path, media_type="text/html")


if __name__ == "__main__":
    import uvicorn
    uvicorn.run("main:app", host="0.0.0.0", port=8080, reload=True)