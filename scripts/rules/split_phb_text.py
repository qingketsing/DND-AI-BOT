from __future__ import annotations

import re
from pathlib import Path


CLEANED_PATH = Path("/home/qingke/DND-AI-BOT/data/raw/Rules/DnD玩家手册5ePHBv1.6版_cleaned.txt")
CHAPTERS_DIR = Path("/home/qingke/DND-AI-BOT/data/raw/Rules/split/chapters")
ENTRIES_DIR = Path("/home/qingke/DND-AI-BOT/data/raw/Rules/split/entries")

CHAPTER_PATTERN = re.compile(r"^(第\s*([0-9一二三四五六七八九十]+)\s*章[:：]\s*(.+))$", re.MULTILINE)

RACE_HEADINGS = {
    "矮人": "Dwarf",
    "精灵": "Elf",
    "半身人": "Halfling",
    "人类": "Human",
    "龙裔": "Dragonborn",
    "侏儒": "Gnome",
    "半精灵": "Half-Elf",
    "半兽人": "Half-Orc",
    "提夫林": "Tiefling",
}
CLASS_HEADINGS = {
    "野蛮人": "Barbarian",
    "诗人": "Bard",
    "吟游诗人": "Bard",
    "牧师": "Cleric",
    "德鲁伊": "Druid",
    "战士": "Fighter",
    "武僧": "Monk",
    "圣武士": "Paladin",
    "游侠": "Ranger",
    "游荡者": "Rogue",
    "术士": "Sorcerer",
    "邪术师": "Warlock",
    "法师": "Wizard",
}


def split_chapters(cleaned: str) -> dict[str, str]:
    """按大章节拆分清洗后的正文。"""
    matches = list(CHAPTER_PATTERN.finditer(cleaned))
    chapters: dict[str, str] = {}
    for index, match in enumerate(matches):
        start = match.start()
        end = matches[index + 1].start() if index + 1 < len(matches) else len(cleaned)
        chapter_number = normalize_chapter_number(match.group(2))
        chapter_title = normalize_chapter_title(match.group(3))
        file_name = f"chapter-{chapter_number}-{chapter_title}.txt"
        chapters[file_name] = cleaned[start:end].strip() + "\n"
    return chapters


def extract_named_entries(chapter_text: str, headings: dict[str, str], prefix: str) -> dict[str, str]:
    """从章节中按预定义标题抓取高置信条目。"""
    lines = chapter_text.splitlines()
    markers: list[tuple[int, str]] = []
    for index, line in enumerate(lines):
        stripped = line.strip()
        for name, english_name in headings.items():
            if is_entry_heading_line(stripped, name, english_name) and stripped != lines[0].strip():
                markers.append((index, name))
                break

    entries: dict[str, str] = {}
    for marker_index, (start_line, name) in enumerate(markers):
        end_line = markers[marker_index + 1][0] if marker_index + 1 < len(markers) else len(lines)
        body = "\n".join(lines[start_line:end_line]).strip()
        if body:
            entries[f"{prefix}-{name}.txt"] = body + "\n"
    return entries


def is_entry_heading_line(line: str, name: str, english_name: str) -> bool:
    """只把真正的标题行识别为条目起点。"""
    return bool(re.fullmatch(rf"{re.escape(name)}\s+{re.escape(english_name)}", line))


def split_cleaned_text(cleaned: str) -> dict[str, dict[str, str]]:
    """输出章节文件和高置信种族/职业条目文件。"""
    chapters = split_chapters(cleaned)
    entries: dict[str, str] = {}
    for chapter_name, chapter_text in chapters.items():
        if chapter_name.startswith("chapter-02-"):
            entries.update(extract_named_entries(chapter_text, RACE_HEADINGS, "race"))
        elif chapter_name.startswith("chapter-03-"):
            entries.update(extract_named_entries(chapter_text, CLASS_HEADINGS, "class"))
    return {"chapters": chapters, "entries": entries}


def normalize_chapter_number(raw_number: str) -> str:
    """统一章节编号为两位数字。"""
    mapping = {
        "一": "01",
        "二": "02",
        "三": "03",
        "四": "04",
        "五": "05",
        "六": "06",
        "七": "07",
        "八": "08",
        "九": "09",
        "十": "10",
        "十一": "11",
    }
    if raw_number.isdigit():
        return f"{int(raw_number):02d}"
    return mapping.get(raw_number, raw_number)


def normalize_chapter_title(raw_title: str) -> str:
    """只保留章节中文标题部分。"""
    title = raw_title.strip()
    title = re.split(r"\s+[A-Za-z]", title, maxsplit=1)[0]
    title = re.sub(r"[\\/:*?\"<>|]", "-", title)
    return title


def clear_output_dir(path: Path) -> None:
    """清空旧的输出文本文件。"""
    path.mkdir(parents=True, exist_ok=True)
    for item in path.glob("*.txt"):
        item.unlink()


def write_outputs(outputs: dict[str, str], output_dir: Path) -> None:
    """将拆分结果写入目标目录。"""
    for file_name, content in outputs.items():
        (output_dir / file_name).write_text(content, encoding="utf-8")


def main() -> None:
    cleaned = CLEANED_PATH.read_text(encoding="utf-8")
    result = split_cleaned_text(cleaned)
    clear_output_dir(CHAPTERS_DIR)
    clear_output_dir(ENTRIES_DIR)
    write_outputs(result["chapters"], CHAPTERS_DIR)
    write_outputs(result["entries"], ENTRIES_DIR)
    print(CHAPTERS_DIR)
    print(ENTRIES_DIR)


if __name__ == "__main__":
    main()
