# Диаграммы архитектуры — Docker CIS Scanner

---

## 1. C4 — уровень Context (контекст системы)

Система целиком, кто ей пользуется и с чем она общается снаружи.

```mermaid
flowchart TB
    user["Специалист по безопасности / разработчик"]
    system["Docker CIS Scanner (наша система)"]
    docker[("Docker daemon: образы и контейнеры")]
    trivy["Trivy (сканер CVE)"]

    user -->|"запускает скан, смотрит отчёт"| system
    system -->|"читает конфигурацию (Docker API)"| docker
    system -->|"запрашивает уязвимости"| trivy
```

---

## 2. C4 — уровень Container (крупные части системы)

Из чего система состоит внутри и по каким протоколам части общаются.

```mermaid
flowchart TB
    user["Пользователь"]
    subgraph sys["Docker CIS Scanner"]
        dash["Streamlit — дашборд (UI)"]
        orch["FastAPI — оркестратор"]
        scanner["Go — сканер-сервис (Gin)"]
    end
    docker[("Docker daemon")]
    trivy["Trivy (CLI)"]

    user -->|"HTTP (браузер)"| dash
    dash -->|"HTTP / REST"| orch
    orch -->|"HTTP / REST"| scanner
    orch -->|"вызов CLI"| trivy
    scanner -->|"Docker Engine API"| docker
```

---

## 3. Диаграмма последовательности — сценарий одного скана

Порядок обмена сообщениями во времени.

```mermaid
sequenceDiagram
    actor U as Пользователь
    participant D as Streamlit
    participant O as FastAPI
    participant G as Go-сканер
    participant T as Trivy
    participant K as Docker daemon

    U->>D: вводит цель, жмёт "Сканировать"
    D->>O: POST /scan {target, with_cve}
    O->>G: POST /scan/image {image}
    G->>K: inspect / history (Docker API)
    K-->>G: данные образа/контейнера
    G-->>O: CIS-отчёт (JSON)
    O->>T: trivy image <target>
    T-->>O: список CVE
    O->>O: слияние CIS + CVE, расчёт оценки
    O-->>D: объединённый результат + scan_id
    D-->>U: сводка и графики
    U->>D: "Скачать отчёт"
    D->>O: GET /report/{scan_id}
    O-->>D: HTML-отчёт
```

---

## 4. Компонентная диаграмма — внутреннее устройство Go-сервиса

```mermaid
flowchart LR
    http["HTTP-слой (Gin): обработчики эндпоинтов"]
    collector["Сборщик данных (Docker client)"]
    engine["Движок проверок (20 CIS-проверок)"]
    agg["Агрегатор: сводка и оценка"]
    docker[("Docker daemon")]

    http --> collector
    collector -->|"Docker API"| docker
    collector --> engine
    engine --> agg
    agg --> http
```
