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

st.set_page_config(page_title="Docker CIS Scanner", layout="wide")

STATUS_ORDER = {"fail": 0, "warn": 1, "pass": 2, "error": 3}


@st.cache_data(ttl=10)
def fetch_health() -> dict:
    try:
        return requests.get(f"{ORCHESTRATOR_URL}/health", timeout=3).json()
    except Exception:
        return {"status": "error", "go_scanner": "offline"}


def show_history():
    try:
        scans = requests.get(f"{ORCHESTRATOR_URL}/scans", timeout=5).json()
    except Exception:
        st.warning("Не удалось загрузить историю.")
        return

    if not scans:
        st.write("История сканов пуста.")
        return

    st.subheader("Выполненные сканы")
    for s in reversed(scans):
        col_name, col_score, col_cve, col_btn = st.columns([3, 1, 1, 2])
        col_name.write(f"{s['target']['type']} / {s['target']['name']}")
        col_score.write(f"{s['cis_score']}/100")
        col_cve.write(f"CVE: {s['cve_count']}")
        col_btn.link_button("Открыть отчёт",
                            f"{PUBLIC_ORCHESTRATOR_URL}/report/{s['scan_id']}")


# ── Страница ────────────────────────────────────────────────────────────────

st.title("Docker CIS Scanner")
st.caption("Анализ безопасности Docker-образов и контейнеров")

health = fetch_health()
if health.get("status") != "ok":
    st.error("Оркестратор недоступен. Запусти: uvicorn main:app --port 8080")
    st.stop()
if health.get("go_scanner") == "offline":
    st.warning("Go-сервис недоступен. Запусти: cd scanner && go run .")

# ── Форма ────────────────────────────────────────────────────────────────────

with st.sidebar:
    st.header("Новый скан")
    target_type = st.selectbox("Тип", ["image", "container"])
    target = st.text_input("Имя образа или контейнера",
                           value="mysql:8.0" if target_type == "image" else "nats")
    with_cve = st.checkbox("Включить CVE-скан (Trivy)", value=True)
    scan_btn = st.button("Запустить", type="primary")

# ── session_state ─────────────────────────────────────────────────────────────

if "report" not in st.session_state:
    st.session_state.report = None
if "scan_error" not in st.session_state:
    st.session_state.scan_error = None

# ── Запуск скана ──────────────────────────────────────────────────────────────

if scan_btn and target:
    st.session_state.scan_error = None
    with st.spinner("Сканирование..."):
        try:
            resp = requests.post(
                f"{ORCHESTRATOR_URL}/scan",
                json={"target_type": target_type, "target": target, "with_cve": with_cve},
                timeout=300,
            )
            if resp.status_code == 200:
                st.session_state.report = resp.json()
            else:
                st.session_state.scan_error = resp.json().get("detail", "Ошибка")
        except requests.ConnectionError:
            st.session_state.scan_error = "Оркестратор недоступен"

if st.session_state.scan_error:
    st.error(st.session_state.scan_error)

# ── Результаты ────────────────────────────────────────────────────────────────

if st.session_state.report:
    r = st.session_state.report
    cis = r["cis_summary"]

    if r.get("overwritten"):
        st.info(f"Результат обновлён: {r['target']['type']} / {r['target']['name']}")
    else:
        st.success(f"Готово: {r['target']['type']} / {r['target']['name']}")

    tab_cis, tab_cve, tab_hist = st.tabs(["CIS-проверки", "CVE", "История"])

    with tab_cis:
        st.write(f"Оценка: **{cis['score']}/100** "
                 f"| Пройдено: {cis['passed']} "
                 f"| Провалено: {cis['failed']} "
                 f"| Предупреждений: {cis['warned']}")
        st.divider()

        checks = sorted(r["cis_report"]["checks"],
                        key=lambda c: STATUS_ORDER.get(c["status"], 4))
        rows = [{
            "Статус":       c["status"].upper(),
            "Проверка":     c["title"],
            "CIS":          c.get("cis_reference") or "—",
            "Серьёзность":  c["severity"].upper(),
            "Детали":       c["details"],
            "Рекомендация": c["remediation"] if c["status"] in ("fail", "warn") else "",
        } for c in checks]
        st.dataframe(rows, height=360)

        col1, col2 = st.columns(2)
        col1.link_button("Открыть HTML-отчёт",
                         f"{PUBLIC_ORCHESTRATOR_URL}/report/{r['scan_id']}")
        try:
            html_bytes = requests.get(
                f"{ORCHESTRATOR_URL}/report/{r['scan_id']}", timeout=10
            ).content
            safe_name = r["target"]["name"].replace(":", "_").replace("/", "_")
            col2.download_button("Скачать HTML-отчёт", html_bytes,
                                 file_name=f"report_{safe_name}.html",
                                 mime="text/html")
        except Exception:
            pass

    with tab_cve:
        vulns = r["vulnerabilities"]
        vs = r["vuln_summary"]
        if not vulns:
            st.write("CVE-уязвимостей не обнаружено.")
        else:
            st.write(f"Всего CVE: **{len(vulns)}** "
                     f"| Critical: {vs['critical']} "
                     f"| High: {vs['high']} "
                     f"| Medium: {vs['medium']} "
                     f"| Low: {vs['low']}")
            st.divider()
            rows = [{
                "CVE":          v["cve"],
                "Пакет":        v["package"],
                "Версия":       v["installed_version"],
                "Исправлено в": v.get("fixed_version") or "—",
                "Серьёзность":  v["severity"].upper(),
            } for v in vulns]
            st.dataframe(rows, height=360)

    with tab_hist:
        show_history()

else:
    st.write("Введи имя образа в боковой панели и нажми «Запустить».")
    show_history()