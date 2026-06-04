"""
Streamlit-дашборд для Docker CIS Scanner
Запуск: streamlit run dashboard.py --server.port 8501
Требует: Go-сервис (:8000) и FastAPI-оркестратор (:8080)
"""
import os
import requests
import streamlit as st

ORCHESTRATOR_URL = os.environ.get("ORCHESTRATOR_URL", "http://localhost:8080")
PUBLIC_ORCHESTRATOR_URL = os.environ.get("PUBLIC_ORCHESTRATOR_URL", ORCHESTRATOR_URL)

st.set_page_config(
    page_title="Docker CIS Scanner",
    page_icon=None,
    layout="wide",
)

STATUS_ICON = {
    "pass": "PASS", "fail": "FAIL",
    "warn": "WARN", "error": "ERROR",
}
SEVERITY_ICON = {
    "critical": "CRITICAL", "high": "HIGH",
    "medium":   "MEDIUM",   "low":  "LOW",
}
STATUS_ORDER = {"fail": 0, "warn": 1, "pass": 2, "error": 3}


# Вспомогательные функции

@st.cache_data(ttl=10)
def fetch_health() -> dict:
    """Проверяет доступность оркестратора (кэш 10 сек)."""
    try:
        return requests.get(f"{ORCHESTRATOR_URL}/health", timeout=3).json()
    except Exception:
        return {"status": "error", "go_scanner": "offline"}


def show_history():
    """Список выполненных сканов из GET /scans."""
    try:
        scans = requests.get(f"{ORCHESTRATOR_URL}/scans", timeout=5).json()
    except Exception:
        st.warning("Не удалось загрузить историю.")
        return

    if not scans:
        st.caption("История сканов пуста.")
        return

    st.subheader("Выполненные сканы")
    for s in reversed(scans):
        c1, c2, c3, c4 = st.columns([3, 1, 1, 2])
        c1.write(f"**{s['target']['type']}** — {s['target']['name']}")
        c2.write(f"Оценка: **{s['cis_score']}/100**")
        c3.write(f"CVE: **{s['cve_count']}**")
        c4.link_button(
            "HTML-отчёт",
            f"{PUBLIC_ORCHESTRATOR_URL}/report/{s['scan_id']}",
        )


# Заголовок и статус

st.title("Docker CIS Scanner")
st.caption("Анализ безопасности Docker-образов · CIS Benchmarks + CVE (Trivy)")

health = fetch_health()
if health.get("status") != "ok":
    st.error("Оркестратор недоступен. Запусти: `uvicorn main:app --port 8080`")
    st.stop()
elif health.get("go_scanner") == "offline":
    st.warning("Go-сервис недоступен. Запусти: `cd scanner && go run .`")

# Боковая панель

with st.sidebar:
    st.header("Новый скан")
    target_type = st.selectbox("Тип цели", ["image", "container"])
    default = "mysql:8.0" if target_type == "image" else "nats"
    target = st.text_input("Имя образа или контейнера", value=default)
    with_cve = st.checkbox("CVE-скан (Trivy)", value=True)
    scan_btn = st.button("Запустить скан", type="primary")
    st.divider()
    st.caption(f"Оркестратор: {ORCHESTRATOR_URL}")

# Инициализация session_state

if "report" not in st.session_state:
    st.session_state.report = None
if "scan_error" not in st.session_state:
    st.session_state.scan_error = None

# Запуск скана

if scan_btn and target:
    st.session_state.scan_error = None
    with st.spinner(f"Сканирование {target}..."):
        try:
            resp = requests.post(
                f"{ORCHESTRATOR_URL}/scan",
                json={"target_type": target_type, "target": target, "with_cve": with_cve},
                timeout=300,
            )
            if resp.status_code == 200:
                st.session_state.report = resp.json()
            else:
                st.session_state.scan_error = resp.json().get("detail", "Ошибка скана")
        except requests.ConnectionError:
            st.session_state.scan_error = "Оркестратор недоступен"

if st.session_state.scan_error:
    st.error(f"Ошибка: {st.session_state.scan_error}")

# Отображение результатов

if st.session_state.report:
    r = st.session_state.report
    cis = r["cis_summary"]
    st.success(f"**{r['target']['type']} — {r['target']['name']}**")

    tab_cis, tab_cve, tab_hist = st.tabs(
        ["CIS-проверки", "CVE-уязвимости", "История"]
    )

    with tab_cis:
        col1, col2, col3, col4 = st.columns(4)
        col1.metric("CIS-оценка", f"{cis['score']}/100")
        col2.metric("Пройдено",       cis["passed"])
        col3.metric("Провалено",      cis["failed"])
        col4.metric("Предупреждений", cis["warned"])
        st.divider()

        checks = sorted(r["cis_report"]["checks"],
                        key=lambda c: STATUS_ORDER.get(c["status"], 4))
        rows = [{
            "Статус":       STATUS_ICON.get(c["status"], c["status"]),
            "Проверка":     c["title"],
            "CIS":          c.get("cis_reference") or "—",
            "Серьёзность":  SEVERITY_ICON.get(c["severity"], c["severity"]),
            "Детали":       c["details"],
            "Рекомендация": c["remediation"] if c["status"] in ("fail", "warn") else "",
        } for c in checks]
        st.dataframe(rows, height=380)

        btn_col1, btn_col2 = st.columns([1, 1])
        btn_col1.link_button("Открыть HTML-отчёт",
                             f"{PUBLIC_ORCHESTRATOR_URL}/report/{r['scan_id']}")
        try:
            html_bytes = requests.get(
                f"{ORCHESTRATOR_URL}/report/{r['scan_id']}", timeout=10
            ).content
            safe_name = r["target"]["name"].replace(":", "_").replace("/", "_")
            btn_col2.download_button(
                label="Скачать HTML-отчёт",
                data=html_bytes,
                file_name=f"report_{safe_name}.html",
                mime="text/html",
            )
        except Exception:
            pass

    with tab_cve:
        vs = r["vuln_summary"]
        vulns = r["vulnerabilities"]
        if not vulns:
            st.info("CVE-уязвимостей не обнаружено.")
        else:
            vc1, vc2, vc3, vc4 = st.columns(4)
            vc1.metric("Critical", vs["critical"])
            vc2.metric("High",     vs["high"])
            vc3.metric("Medium",   vs["medium"])
            vc4.metric("Low",      vs["low"])
            st.divider()
            vuln_rows = [{
                "CVE":          v["cve"],
                "Пакет":        v["package"],
                "Версия":       v["installed_version"],
                "Исправлено в": v.get("fixed_version") or "—",
                "Серьёзность":  SEVERITY_ICON.get(v["severity"], v["severity"]),
            } for v in vulns]
            st.dataframe(vuln_rows, height=380)

    with tab_hist:
        show_history()

else:
    st.info("Введи имя образа в боковой панели и нажми «Запустить скан»")
    show_history()