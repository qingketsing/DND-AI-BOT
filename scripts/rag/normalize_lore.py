from __future__ import annotations

import re
import sys
from pathlib import Path

if __package__ is None or __package__ == "":
    sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from scripts.rag.common_models import NormalizedDocument
from scripts.rag.io_utils import clear_json_outputs, save_json


ROOT = Path("/home/qingke/DND-AI-BOT")
LORE_SOURCE_PATH = ROOT / "data/raw/Background/settings.md"
NORMALIZED_DIR = ROOT / "data/normalized/lore"

NOISE_HEADINGS = {
    "城市",
    "不止于此的异世界",
    "图片鸣谢",
    "字体鸣谢",
    "兼容",
    "我们贩卖铅弹",
    "火车",
}

TITLE_SLUGS = {
    "死寂的世界": "dead-world",
    "空旷的大厅": "empty-halls",
    "但并非完全死寂……": "not-entirely-dead",
    "凝固的天空": "frozen-sky",
    "通往城市的门": "gates-to-the-city",
    "地下的世界": "world-below",
    "暮光大厅": "twilight-halls",
    "车站": "stations",
    "古老的巨兽": "ancient-beasts",
    "轨道": "rails",
    "迷失与需要帮助者": "the-lost-and-the-needy",
    "无光之厅": "lightless-halls",
    "之下": "beneath",
    "无光之生": "lightless-life",
}


def strip_lore_noise(markdown_text: str) -> str:
    """删除图片、封面页和版权类噪音，仅保留内容章节。"""
    text = re.sub(r"!\[image\]\([^)]+\)\n?", "", markdown_text)
    text = text.replace("\r\n", "\n").replace("\r", "\n")
    lines = text.splitlines()
    kept: list[str] = []
    skipping = True

    for line in lines:
        stripped = line.strip()
        if stripped.startswith("# "):
            title = stripped[2:].strip()
            if title == "死寂的世界":
                skipping = False
            if title in NOISE_HEADINGS and title != "死寂的世界":
                continue
        if skipping:
            continue
        kept.append(line)

    text = "\n".join(kept)
    text = re.sub(r"\n{3,}", "\n\n", text).strip() + "\n"
    return text


def split_markdown_sections(markdown_text: str) -> list[dict]:
    """按一级标题拆成内容章节。"""
    sections: list[dict] = []
    current_title: str | None = None
    current_lines: list[str] = []

    for raw_line in markdown_text.splitlines():
        stripped = raw_line.strip()
        if stripped.startswith("# "):
            title = stripped[2:].strip()
            if title in NOISE_HEADINGS and title != "死寂的世界":
                continue
            if current_title is not None:
                sections.append({"title": current_title, "content": "\n".join(current_lines).strip()})
            current_title = title
            current_lines = []
            continue
        if current_title is not None:
            current_lines.append(stripped)

    if current_title is not None:
        sections.append({"title": current_title, "content": "\n".join(current_lines).strip()})

    return [section for section in sections if section["content"]]


def build_lore_document_id(setting_id: str, title: str) -> str:
    suffix = TITLE_SLUGS.get(title, slugify(title))
    return f"lore:{setting_id}:{suffix}"


def infer_lore_tags(title: str, content: str) -> list[str]:
    _ = content
    return ["setting_section", slugify(title)]


def build_lore_documents(sections: list[dict], source_file: str, setting_id: str) -> list[NormalizedDocument]:
    documents: list[NormalizedDocument] = []
    for section in sections:
        title = section["title"]
        documents.append(
            NormalizedDocument(
                id=build_lore_document_id(setting_id, title),
                knowledge_base="lore",
                source_type="background_md",
                source_file=source_file,
                title=title,
                doc_type="setting_section",
                language="zh",
                content=section["content"],
                setting_id=setting_id,
                section_path=[title],
                tags=infer_lore_tags(title, section["content"]),
                aliases=[title],
            )
        )
    return documents


def normalize_lore(markdown_path: Path, output_dir: Path, setting_id: str = "default-setting") -> list[NormalizedDocument]:
    markdown_text = markdown_path.read_text(encoding="utf-8")
    cleaned = strip_lore_noise(markdown_text)
    sections = split_markdown_sections(cleaned)
    documents = build_lore_documents(sections, markdown_path.name, setting_id)
    clear_json_outputs(output_dir)
    for document in documents:
        save_json(output_dir / f"{document.id.replace(':', '-')}.json", document.to_dict())
    return documents


def slugify(value: str) -> str:
    lowered = value.lower().strip()
    lowered = re.sub(r"[^a-z0-9\u4e00-\u9fff]+", "-", lowered)
    lowered = re.sub(r"-{2,}", "-", lowered).strip("-")
    return lowered


def main() -> None:
    normalize_lore(LORE_SOURCE_PATH, NORMALIZED_DIR)
    print(NORMALIZED_DIR)


if __name__ == "__main__":
    main()
