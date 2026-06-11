# Docker CIS Scanner

[![CI](https://github.com/shmeoww/docker-cis-benchmarks/actions/workflows/ci.yml/badge.svg)](https://github.com/shmeoww/docker-cis-benchmarks/actions/workflows/ci.yml)

Инструмент автоматического анализа безопасности Docker-образов и контейнеров по стандарту **CIS Docker Benchmark v1.8.0** с CVE-сканированием через Trivy.

**Курсовая работа** по дисциплине «Методы и технологии программирования», Вариант 6.

---

## Возможности

- **20 CIS-проверок** — 6 для образов и 14 для контейнеров (привилегированный режим, capabilities, docker.sock, ресурсные лимиты и др.)
- **CVE-сканирование** — поиск уязвимых пакетов внутри образа через Trivy
- **HTML-отчёт** — скачивается или открывается в браузере
- **Параллельное выполнение** — все проверки запускаются одновременно (горутины)
- **История сканов** — сохраняется между перезапусками
- **Streamlit-дашборд** и REST API с автодокументацией (Swagger/OpenAPI)

---

## Стек

| Компонент | Технология |
|---|---|
| Go-сервис (CIS-проверки) | Go 1.26, Gin, moby/moby SDK v0.3.0 |
| Python-оркестратор | Python 3.11, FastAPI, Pydantic V2 |
| CVE-скан | Trivy 0.71 |
| Дашборд | Streamlit |
| Отчёты | Jinja2 (HTML) |
| Контейнеризация | Docker, Docker Compose |
| CI | GitHub Actions, gosec, bandit, govulncheck, pip-audit |

---

## Архитектура

```
┌─────────────────────────────────────────────────────┐
│                  Пользователь                        │
│         браузер: http://localhost:8501               │
└───────────────────────┬─────────────────────────────┘
                        │
                        ▼
┌───────────────────────────────────────┐
│         Streamlit (порт 8501)         │
│   дашборд: форма скана, таблицы,      │
│   история, скачивание отчёта          │
└───────────────────────┬───────────────┘
                        │ HTTP POST /scan
                        ▼
┌───────────────────────────────────────┐    ┌─────────────┐
│    FastAPI оркестратор (порт 8080)    │    │   Trivy CLI │
│                                       │───▶│  CVE-скан   │
│  • /scan    • /scans   • /report      │    │  (подпроцесс│
│  • история сканов в data/history.json │    │  subprocess)│
└───────────────────────┬───────────────┘    └─────────────┘
                        │ HTTP POST /scan/image
                        ▼
┌───────────────────────────────────────┐
│      Go-сервис (порт 8000)            │
│                                       │
│  • 20 CIS-проверок (горутины)         │
│  • Docker SDK → инспекция образов     │
│    и контейнеров                      │
└───────────────────────┬───────────────┘
                        │ Docker API (unix socket)
                        ▼
              Docker daemon (хост)
```

### Как работает скан (шаг за шагом)

1. Пользователь вводит имя образа в дашборде и нажимает «Запустить»
2. Streamlit отправляет `POST /scan` в FastAPI-оркестратор
3. Оркестратор параллельно:
   - Вызывает Go-сервис `POST /scan/image` → CIS-проверки
   - Запускает `trivy image --format json` → CVE-уязвимости
4. Go-сервис подключается к Docker через SDK, инспектирует образ (`ImageInspect`, `ImageHistory`), затем запускает все 6 проверок **одновременно** через горутины и каналы
5. Оркестратор сливает результаты CIS + CVE в `UnifiedReport`, сохраняет в `history.json` и генерирует HTML-отчёт
6. Дашборд отображает таблицы с результатами и кнопки для открытия/скачивания отчёта

---

## CIS-проверки

### Образы (раздел 4)

| ID | Проверка | CIS | Severity |
|---|---|---|---|
| `image_no_root_user` | USER не root | 4.1 | Critical |
| `image_has_healthcheck` | HEALTHCHECK задан | 4.6 | Medium |
| `image_use_copy_not_add` | COPY вместо ADD | 4.9 | Low |
| `image_no_latest_tag` | Версия зафиксирована | — | Medium |
| `image_no_secrets_in_env` | Нет секретов в ENV | 4.10 | High |
| `image_no_privileged_ports` | Нет SSH/привилегированных портов | 5.8 | High |

### Контейнеры (раздел 5)

| ID | Проверка | CIS | Severity |
|---|---|---|---|
| `container_no_privileged` | Не --privileged | 5.5 | Critical |
| `container_restricted_capabilities` | Нет опасных capabilities | 5.4 | High |
| `container_no_host_network` | Не --net=host | 5.10 | High |
| `container_no_host_pid` | Не --pid=host | 5.16 | High |
| `container_no_host_ipc` | Не --ipc=host | 5.17 | Medium |
| `container_no_docker_socket` | docker.sock не примонтирован | 5.32 | Critical |
| `container_no_new_privileges` | no-new-privileges установлен | 5.26 | High |
| `container_readonly_rootfs` | Корневая ФС read-only | 5.13 | Medium |
| `container_memory_limit` | Лимит памяти задан | 5.11 | Medium |
| `container_pids_limit` | Лимит PIDs задан | 5.29 | Low |
| `container_cpu_limit` | Лимит CPU задан | 5.12 | Medium |
| `container_restart_policy` | Безопасная политика перезапуска | 5.15 | Low |
| `container_no_sensitive_mounts` | Нет монтирования /, /etc, /boot | 5.6 | Critical |
| `container_seccomp_apparmor` | seccomp/AppArmor не отключены | 5.22 | High |

---

## API

Полная интерактивная документация: **http://localhost:8080/docs**

### POST /scan — запустить скан

```json
// Запрос
{
  "target_type": "image",      // "image" или "container"
  "target":      "mysql:8.0",  // имя образа или ID/имя контейнера
  "with_cve":    true          // false = только CIS-проверки (быстрее)
}

// Ответ
{
  "scan_id":    "image_mysql_8.0",
  "scanned_at": "2026-06-03T14:30:00Z",
  "target":     { "type": "image", "id": "sha256:...", "name": "mysql:8.0" },
  "cis_summary": { "total": 6, "passed": 4, "failed": 2, "score": 66 },
  "cis_report": { "checks": [...] },
  "vulnerabilities": [...],
  "vuln_summary": { "critical": 1, "high": 17 },
  "overwritten": false
}
```

### GET /scans — история сканов

```json
// Ответ — список кратких записей
[
  {
    "scan_id":    "image_mysql_8.0",
    "target":     { "type": "image", "name": "mysql:8.0" },
    "scanned_at": "2026-06-03T14:30:00Z",
    "cis_score":  66,
    "cve_count":  46
  }
]
```

### GET /scans/{scan_id} — полный отчёт по ID

Возвращает полный `UnifiedReport` (тот же формат что `/scan`).

### GET /report/{scan_id} — HTML-отчёт

Возвращает готовый HTML-файл, открывается прямо в браузере.

### GET /health — статус сервисов

```json
{ "status": "ok", "go_scanner": "online" }
```

---

## Быстрый старт

**Требования:** Docker Desktop

```bash
git clone https://github.com/shmeoww/docker-cis-benchmarks.git
cd docker-cis-benchmarks
docker compose up --build
```

| Сервис | URL |
|---|---|
| Дашборд (Streamlit) | http://localhost:8501 |
| API + Swagger | http://localhost:8080/docs |
| Go-сервис | http://localhost:8000/health |

---

## Разработка (локально, без Docker)

**Требования:** Go 1.26, Python 3.11, Trivy, Docker Desktop

```bash
# 1. Go-сервис
cd scanner && go run .

# 2. Оркестратор (новый терминал)
cd orchestrator
.venv\Scripts\Activate.ps1
uvicorn main:app --port 8080

# 3. Дашборд (новый терминал)
cd orchestrator
streamlit run dashboard.py --server.port 8501
```

---

## Тесты

### Подготовка (для интеграционных тестов)

Некоторые Go-тесты проверяют реальное сканирование и требуют локальных образов и запущенного контейнера. Без них тест **пропустится** (SKIP), а не упадёт (FAIL).

Оба образа и базовый образ для контейнера — **официальные публичные образы** с Docker Hub / Google Container Registry, доступны без авторизации.

```bash
# Образы для тестов сканирования образов
docker pull mysql:8.0                               # TestScanImage, TestScanImageEndpoint
docker pull mirror.gcr.io/library/nats:2.10-alpine  # TestCollectImage

# Контейнер для тестов сканирования контейнеров
# TestCollectContainer, TestScanContainer, TestScanAll берут первый доступный контейнер
docker run -d --name cis-test-container alpine sleep infinity
```

> Остановить и удалить тестовый контейнер после работы:
> ```bash
> docker rm -f cis-test-container
> ```

### Запуск

```bash
# Go — unit-тесты (без Docker-образов, запускаются везде)
cd scanner
go test -short ./internal/...

# Go — все тесты включая интеграционные (нужны образы выше)
go test -v ./...

# Go — покрытие (85%)
go test -coverprofile="coverage.out" ./...
go tool cover -func="coverage.out"

# Python — все тесты (43 теста, 93% покрытие)
cd orchestrator && pytest -v

# Python — только unit (без Go-сервиса, запускаются везде)
pytest -k "not Integration" -v
```

### Уровни тестов

| Уровень | Где | Зависимости | Запуск в CI |
|---|---|---|---|
| Unit (Go) | `internal/checks`, `internal/model` | нет | ✅ |
| Unit (Python) | `tests/test_merger`, `test_trivy_client` | нет | ✅ |
| Интеграционные (Go) | `main_test.go`, `docker_test.go` | Docker + образы | ⏭ пропускаются |
| Интеграционные (Python) | `TestIntegration` class | Go-сервис | ⏭ пропускаются |

---

## Структура проекта

```
docker-cis-benchmarks/
├── scanner/                        Go-сервис
│   ├── internal/
│   │   ├── checks/                 20 CIS-проверок + движок (горутины/каналы)
│   │   ├── docker/                 сборщики: ImageData, ContainerData
│   │   └── model/                  типы данных, JSON-контракт
│   ├── main.go                     Gin HTTP-сервер
│   └── Dockerfile                  multi-stage сборка (~15 МБ)
├── orchestrator/                   Python-сервис
│   ├── main.py                     FastAPI-эндпоинты
│   ├── merger.py                   оркестрация CIS + CVE
│   ├── scanner_client.py           HTTP-клиент к Go
│   ├── trivy_client.py             CVE-скан через Trivy CLI
│   ├── storage.py                  персистентная история (JSON)
│   ├── dashboard.py                Streamlit UI
│   ├── templates/report.html       HTML-шаблон отчёта (Jinja2)
│   └── tests/                      43 теста, 93% покрытие
├── docker-compose.yml              три сервиса в одной сети
├── .github/workflows/ci.yml        GitHub Actions CI/CD
└── requirements.txt
```

---

## ИИ-инструменты в разработке
 
При работе над проектом использовался **Claude (Anthropic)** как AI-ассистент.
 
Основные сценарии применения:
- изучение документации (Docker Engine API, moby/moby SDK, CIS Docker Benchmark v1.8.0) и объяснение незнакомых концепций
- помощь в выборе технологий: обсуждение связки Go + Python, сравнение подходов к интеграции (HTTP vs subprocess vs gRPC)
- разбор примеров кода на Go — работа с горутинами, каналами, обработка ошибок Docker API
- помощь при написании тестов: table-driven тесты на Go, pytest-фикстуры, моки для изоляции компонентов
- оформление документации (README, комментарии в коде)
Все архитектурные решения, реализация логики проверок и итоговый код проверялись и дорабатывались самостоятельно.
 
---

## Лицензия

[MIT](LICENSE)