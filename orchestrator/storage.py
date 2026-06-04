"""
Хранилище истории сканов, читает и пишет результаты в data/history.json
"""
import json
from pathlib import Path

from models import UnifiedReport

DATA_DIR = Path("data")
HISTORY_FILE = DATA_DIR / "history.json"


def load_results() -> dict[str, UnifiedReport]:
    """
    Загружает историю сканов из файла при старте приложения
    Возвращает пустой словарь если файл не существует или повреждён
    """
    if not HISTORY_FILE.exists():
        return {}
    try:
        raw = json.loads(HISTORY_FILE.read_text(encoding="utf-8"))
        return {k: UnifiedReport.model_validate(v) for k, v in raw.items()}
    except Exception as e:
        print(f"[storage] не удалось загрузить историю: {e}")
        return {}


def save_results(results: dict[str, UnifiedReport]) -> None:
    """
    Сохраняет все результаты в файл
    Вызывается после каждого нового скана
    """
    DATA_DIR.mkdir(parents=True, exist_ok=True)
    data = {k: json.loads(v.model_dump_json()) for k, v in results.items()}
    HISTORY_FILE.write_text(
        json.dumps(data, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )