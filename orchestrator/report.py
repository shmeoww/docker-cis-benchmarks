"""
Генерация HTML-отчёта через Jinja2-шаблон
"""
import re
from datetime import datetime, timezone
from pathlib import Path

from jinja2 import Environment, FileSystemLoader
from models import UnifiedReport

TEMPLATES_DIR = Path(__file__).parent / "templates"
REPORTS_DIR = Path(__file__).parent / "reports"


def _safe_name(name: str) -> str:
    """
    Превращает имя образа/контейнера в безопасное имя файла
    'mysql:8.0' → 'mysql_8.0'
    '/nats'     → 'nats'
    """
    name = name.lstrip("/")          
    name = re.sub(r"[:/\\]", "_", name) 
    name = re.sub(r"[^\w\-.]", "", name)
    return name or "unknown"


def generate_html_report(report: UnifiedReport) -> str:
    """Принимает UnifiedReport, возвращает готовый HTML как строку"""
    env = Environment(
        loader=FileSystemLoader(str(TEMPLATES_DIR)),
        autoescape=False, 
    )
    template = env.get_template("report.html")
    return template.render(report=report)


def save_html_report(
    report: UnifiedReport,
    output_dir: str | None = None,
) -> str:
    """
    Генерирует HTML и сохраняет в файл
    Если файл с таким именем уже есть — перезаписывается
    Возвращает путь к файлу
    """
    save_dir = Path(output_dir) if output_dir else REPORTS_DIR
    save_dir.mkdir(parents=True, exist_ok=True)

    safe = _safe_name(report.target.name)
    filename = f"report_{report.target.type}_{safe}.html"
    filepath = save_dir / filename

    html = generate_html_report(report)
    filepath.write_text(html, encoding="utf-8")
    return str(filepath)