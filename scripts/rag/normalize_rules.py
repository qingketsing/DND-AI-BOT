from __future__ import annotations

import re
import sys
from pathlib import Path

if __package__ is None or __package__ == "":
    sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from scripts.rag.common_models import NormalizedDocument
from scripts.rag.io_utils import clear_json_outputs, save_json


ROOT = Path("/home/qingke/DND-AI-BOT")
RULES_RAW_DIR = ROOT / "data/raw/Rules"
CHAPTERS_DIR = RULES_RAW_DIR / "split/chapters"
ENTRIES_DIR = RULES_RAW_DIR / "split/entries"
NORMALIZED_DIR = ROOT / "data/normalized/rules"
CLEANED_PATH = RULES_RAW_DIR / "DnD玩家手册5ePHBv1.6版_cleaned.txt"


ENTRY_ENGLISH_NAMES = {
    "法师": "wizard",
    "战士": "fighter",
    "野蛮人": "barbarian",
    "吟游诗人": "bard",
    "牧师": "cleric",
    "德鲁伊": "druid",
    "武僧": "monk",
    "圣武士": "paladin",
    "游侠": "ranger",
    "游荡者": "rogue",
    "术士": "sorcerer",
    "邪术师": "warlock",
    "矮人": "dwarf",
    "精灵": "elf",
    "半身人": "halfling",
    "人类": "human",
    "龙裔": "dragonborn",
    "侏儒": "gnome",
    "半精灵": "half-elf",
    "半兽人": "half-orc",
    "提夫林": "tiefling",
}

CHAPTER_ENGLISH_NAMES = {
    "一步步创建角色": "01-character-creation",
    "种族": "02-races",
    "职业": "03-classes",
    "个性与背景": "04-personality-and-background",
    "装备": "05-equipment",
    "自定义选项": "06-customization-options",
    "属性值应用": "07-using-ability-scores",
    "冒险": "08-adventuring",
    "战斗": "09-combat",
    "施法": "10-spellcasting",
    "法术": "11-spells",
}

ENTRY_SECTION_PATHS = {
    "class": "第 3 章：职业",
    "race": "第 2 章：种族",
}


def infer_rule_doc_type(file_name: str) -> str:
    if file_name.startswith("class-"):
        return "class"
    if file_name.startswith("race-"):
        return "race"
    if file_name.startswith("chapter-"):
        return "chapter"
    raise ValueError(f"unsupported rules file: {file_name}")


def build_rule_document_id(file_name: str) -> str:
    doc_type = infer_rule_doc_type(file_name)
    stem = Path(file_name).stem
    if doc_type == "chapter":
        _, chapter_no, title = stem.split("-", 2)
        suffix = CHAPTER_ENGLISH_NAMES.get(title, f"{chapter_no}-{slugify(title)}")
        return f"rules:phb:chapter:{suffix}"
    _, title = stem.split("-", 1)
    suffix = ENTRY_ENGLISH_NAMES.get(title, slugify(title))
    return f"rules:phb:{doc_type}:{suffix}"


def infer_rule_tags(title: str, doc_type: str) -> list[str]:
    tags = [doc_type]
    english = ENTRY_ENGLISH_NAMES.get(title)
    if english:
        tags.append(english)
    return tags


def build_chapter_documents(chapters_dir: Path) -> list[NormalizedDocument]:
    documents: list[NormalizedDocument] = []
    for path in sorted(chapters_dir.glob("chapter-*.txt")):
        stem = path.stem
        _, chapter_no, title = stem.split("-", 2)
        chapter_title = f"第 {int(chapter_no)} 章：{title}"
        documents.append(
            NormalizedDocument(
                id=build_rule_document_id(path.name),
                knowledge_base="rules",
                source_type="phb",
                source_file=path.name,
                title=title,
                doc_type="chapter",
                language="zh",
                content=path.read_text(encoding="utf-8").strip(),
                chapter=chapter_title,
                book="PHB",
                section_path=[chapter_title],
                tags=["chapter", slugify(title)],
                aliases=[title],
            )
        )
    return documents


def build_entry_documents(entries_dir: Path) -> list[NormalizedDocument]:
    documents: list[NormalizedDocument] = []
    for path in sorted(entries_dir.glob("*.txt")):
        doc_type = infer_rule_doc_type(path.name)
        title = Path(path.name).stem.split("-", 1)[1]
        chapter_title = ENTRY_SECTION_PATHS[doc_type]
        alias = ENTRY_ENGLISH_NAMES.get(title, "")
        aliases = [title]
        if alias:
            aliases.append(alias.title() if "-" not in alias else alias)
        documents.append(
            NormalizedDocument(
                id=build_rule_document_id(path.name),
                knowledge_base="rules",
                source_type="phb",
                source_file=path.name,
                title=title,
                doc_type=doc_type,
                language="zh",
                content=path.read_text(encoding="utf-8").strip(),
                chapter=chapter_title,
                book="PHB",
                section_path=[chapter_title, title],
                tags=infer_rule_tags(title, doc_type),
                aliases=aliases,
            )
        )
    return documents


def normalize_rules(cleaned_path: Path, chapters_dir: Path, entries_dir: Path, output_dir: Path) -> list[NormalizedDocument]:
    _ = cleaned_path
    documents = [*build_chapter_documents(chapters_dir), *build_entry_documents(entries_dir)]
    clear_json_outputs(output_dir)
    for document in documents:
        save_json(output_dir / f"{document.id.replace(':', '-').replace('/', '-')}.json", document.to_dict())
    return documents


def slugify(value: str) -> str:
    lowered = value.lower().strip()
    lowered = re.sub(r"[^a-z0-9\u4e00-\u9fff]+", "-", lowered)
    lowered = re.sub(r"-{2,}", "-", lowered).strip("-")
    return lowered


def main() -> None:
    normalize_rules(CLEANED_PATH, CHAPTERS_DIR, ENTRIES_DIR, NORMALIZED_DIR)
    print(NORMALIZED_DIR)


if __name__ == "__main__":
    main()
