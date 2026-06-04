"""
Тесты FastAPI-эндпоинтов.

Unit-тесты (mock): не требуют Go-сервиса, работают всегда.
Integration-тесты: используют fixture require_go_service — пропускаются
если Go недоступен.
"""
import pytest
from unittest.mock import patch


# Вспомогательная функция

def _scan_payload(target_type="image", target="mysql:8.0", with_cve=False):
    return {"target_type": target_type, "target": target, "with_cve": with_cve}


# GET /health

class TestHealth:

    def test_health_returns_200(self, client):
        resp = client.get("/health")
        assert resp.status_code == 200

    def test_health_has_status_ok(self, client):
        resp = client.get("/health")
        assert resp.json()["status"] == "ok"

    def test_health_has_go_scanner_field(self, client):
        resp = client.get("/health")
        assert "go_scanner" in resp.json()


# POST /scan — валидация

class TestScanValidation:

    def test_missing_target_returns_422(self, client):
        """Pydantic валидирует тело: нет поля target → 422 Unprocessable Entity."""
        resp = client.post("/scan", json={"target_type": "image"})
        assert resp.status_code == 422

    def test_missing_target_type_returns_422(self, client):
        resp = client.post("/scan", json={"target": "mysql:8.0"})
        assert resp.status_code == 422

    def test_invalid_target_type_returns_400(self, client):
        """Неверный target_type обрабатываем вручную → 400."""
        with patch("main.orchestrate_image_scan") as mock_orch:
            resp = client.post("/scan", json=_scan_payload(target_type="invalid"))
        assert resp.status_code == 400
        assert "target_type" in resp.json()["detail"]

    def test_empty_body_returns_422(self, client):
        resp = client.post("/scan", json={})
        assert resp.status_code == 422


# POST /scan — успешный скан (mock)

class TestScanWithMock:

    def test_scan_image_returns_200(self, client, mock_unified_report):
        """Успешный скан образа: мокаем orchestrate_image_scan."""
        with patch("main.orchestrate_image_scan", return_value=mock_unified_report):
            resp = client.post("/scan", json=_scan_payload())
        assert resp.status_code == 200

    def test_scan_returns_unified_report_structure(self, client, mock_unified_report):
        with patch("main.orchestrate_image_scan", return_value=mock_unified_report):
            resp = client.post("/scan", json=_scan_payload())
        data = resp.json()
        assert "scan_id" in data
        assert "cis_report" in data
        assert "vulnerabilities" in data
        assert "cis_summary" in data

    def test_scan_stores_result_in_history(self, client, mock_unified_report):
        """После скана результат доступен через GET /scans/{id}."""
        with patch("main.orchestrate_image_scan", return_value=mock_unified_report):
            scan_resp = client.post("/scan", json=_scan_payload())
        scan_id = scan_resp.json()["scan_id"]

        get_resp = client.get(f"/scans/{scan_id}")
        assert get_resp.status_code == 200
        assert get_resp.json()["scan_id"] == scan_id

    def test_scan_container_calls_container_orchestrator(self, client, mock_unified_report):
        """Для target_type=container вызывается orchestrate_container_scan."""
        with patch("main.orchestrate_container_scan", return_value=mock_unified_report) as mock_orch:
            resp = client.post("/scan", json=_scan_payload(target_type="container", target="my-container"))
        assert resp.status_code == 200
        mock_orch.assert_called_once()


# GET /scans

class TestListScans:

    def test_list_scans_returns_200(self, client):
        resp = client.get("/scans")
        assert resp.status_code == 200

    def test_list_scans_returns_list(self, client):
        resp = client.get("/scans")
        assert isinstance(resp.json(), list)

    def test_scan_appears_in_list_after_scan(self, client, mock_unified_report):
        with patch("main.orchestrate_image_scan", return_value=mock_unified_report):
            client.post("/scan", json=_scan_payload())
        resp = client.get("/scans")
        assert len(resp.json()) >= 1
        ids = [item["scan_id"] for item in resp.json()]
        assert mock_unified_report.scan_id in ids


# GET /scans/{id}

class TestGetScan:

    def test_nonexistent_scan_returns_404(self, client):
        resp = client.get("/scans/nonexistent-id")
        assert resp.status_code == 404

    def test_existing_scan_returns_200(self, client, mock_unified_report):
        with patch("main.orchestrate_image_scan", return_value=mock_unified_report):
            client.post("/scan", json=_scan_payload())
        resp = client.get(f"/scans/{mock_unified_report.scan_id}")
        assert resp.status_code == 200


# GET /report/{id}

class TestGetReport:

    def test_nonexistent_report_returns_404(self, client):
        resp = client.get("/report/nonexistent-id")
        assert resp.status_code == 404

    def test_existing_report_returns_html(self, client, mock_unified_report):
        with patch("main.orchestrate_image_scan", return_value=mock_unified_report):
            client.post("/scan", json=_scan_payload())
        resp = client.get(f"/report/{mock_unified_report.scan_id}")
        assert resp.status_code == 200
        assert "text/html" in resp.headers["content-type"]
        assert b"<!DOCTYPE html>" in resp.content


# Интеграционные тесты (требуют Go-сервис)

class TestIntegration:

    def test_real_image_scan(self, client, require_go_service):
        """Полный скан mysql:8.0 с реальным Go-сервисом."""
        resp = client.post("/scan", json=_scan_payload(with_cve=False))
        assert resp.status_code == 200
        data = resp.json()
        assert data["cis_report"]["target"]["name"] == "mysql:8.0"
        assert len(data["cis_report"]["checks"]) == 6
        assert data["cis_summary"]["score"] > 0

    def test_real_scan_then_get_by_id(self, client, require_go_service):
        """Скан + получение по id."""
        scan_resp = client.post("/scan", json=_scan_payload(with_cve=False))
        assert scan_resp.status_code == 200
        scan_id = scan_resp.json()["scan_id"]

        get_resp = client.get(f"/scans/{scan_id}")
        assert get_resp.status_code == 200
        assert get_resp.json()["scan_id"] == scan_id

    def test_real_html_report(self, client, require_go_service):
        """Полный цикл: скан → HTML-отчёт через эндпоинт."""
        scan_resp = client.post("/scan", json=_scan_payload(with_cve=False))
        scan_id = scan_resp.json()["scan_id"]
        report_resp = client.get(f"/report/{scan_id}")
        assert report_resp.status_code == 200
        assert b"CIS" in report_resp.content