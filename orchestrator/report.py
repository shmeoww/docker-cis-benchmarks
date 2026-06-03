"""
Генерация HTML-отчёта через Jinja2-шаблон.
"""
import re
from datetime import datetime, timezone
from pathlib import Path

from jinja2 import Environment, FileSystemLoader
from models import UnifiedReport

# Папка с шаблонами — рядом с этим файлом
TEMPLATES_DIR = Path(__file__).parent / "templates"
# Папка для сохранения готовых отчётов
REPORTS_DIR = Path(__file__).parent / "reports"


def _safe_name(name: str) -> str:
    """
    Превращает имя образа/контейнера в безопасное имя файла.
    'mysql:8.0' → 'mysql_8.0'
    '/nats'     → 'nats'
    """
    name = name.lstrip("/")          # убираем ведущий слеш у контейнеров
    name = re.sub(r"[:/\\]", "_", name)  # : / \ → _
    name = re.sub(r"[^\w\-.]", "", name) # убираем остальные спецсимволы
    return name or "unknown"


def generate_html_report(report: UnifiedReport) -> str:
    """Принимает UnifiedReport, возвращает готовый HTML как строку."""
    env = Environment(
        loader=FileSystemLoader(str(TEMPLATES_DIR)),
        autoescape=False,   # без экранирования: у нас нет пользовательского ввода
    )
    template = env.get_template("report.html")
    return template.render(report=report)


def save_html_report(
    report: UnifiedReport,
    output_dir: str | None = None,
) -> str:
    """
    Генерирует HTML и сохраняет в файл.

    Имя файла: report_{тип}_{имя}.html
      Пример:  report_image_mysql_8.0.html
               report_container_nats.html

    Если файл с таким именем уже есть — перезаписывается.
    Возвращает путь к файлу.
    """
    save_dir = Path(output_dir) if output_dir else REPORTS_DIR
    save_dir.mkdir(parents=True, exist_ok=True)  # создать если не существует

    safe = _safe_name(report.target.name)
    filename = f"report_{report.target.type}_{safe}.html"
    filepath = save_dir / filename

    html = generate_html_report(report)
    filepath.write_text(html, encoding="utf-8")
    return str(filepath)