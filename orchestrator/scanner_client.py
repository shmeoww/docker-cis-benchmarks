"""
HTTP-клиент для общения с Go-сервисом сканера.
Go-сервис должен быть запущен на SCANNER_URL (по умолчанию localhost:8000).
"""
import os
import requests

from models import ScanReport


# URL Go-сервиса берётся из переменной окружения, либо используется дефолт.
# В docker-compose это будет http://scanner:8000 (имя сервиса в сети).
_DEFAULT_URL = "http://localhost:8000"


class ScannerClient:
    """Клиент к Go-сервису сканера."""

    def __init__(self, base_url: str | None = None):
        self.base_url = (base_url
                         or os.environ.get("SCANNER_URL", _DEFAULT_URL))

    # ── Приватный метод: отправить запрос и вернуть JSON ───────────────────
    def _post(self, path: str, payload: dict | None = None) -> dict:
        """
        Отправляет POST-запрос к Go-сервису.
        Возвращает распаршенный JSON или бросает исключение при ошибке.
        """
        resp = requests.post(
            f"{self.base_url}{path}",
            json=payload or {},
            timeout=120,  # сканирование может занять время
        )
        resp.raise_for_status()  # HTTPError при 4xx/5xx
        return resp.json()

    # ── Публичные методы ───────────────────────────────────────────────────

    def scan_image(self, image: str) -> ScanReport:
        """Сканирует один образ, возвращает ScanReport с 6 CIS-проверками."""
        data = self._post("/scan/image", {"image": image})
        return ScanReport.model_validate(data)

    def scan_container(self, container: str) -> ScanReport:
        """Сканирует один контейнер, возвращает ScanReport с 14 проверками."""
        data = self._post("/scan/container", {"container": container})
        return ScanReport.model_validate(data)

    def scan_all(self) -> list[ScanReport]:
        """Сканирует все образы и контейнеры, возвращает список отчётов."""
        data = self._post("/scan/all")
        reports_data = data.get("scans", [])
        return [ScanReport.model_validate(r) for r in reports_data]

    def health(self) -> bool:
        """
        Проверяет доступность Go-сервиса.
        Возвращает True если сервис отвечает, False если нет.
        """
        try:
            resp = requests.get(f"{self.base_url}/health", timeout=5)
            return resp.status_code == 200
        except requests.RequestException:
            return False