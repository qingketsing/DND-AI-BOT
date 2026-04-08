from __future__ import annotations

import json
import re
import sys
from pathlib import Path

if __package__ is None or __package__ == "":
    sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from scripts.rag.common_models import ChunkDocument
from scripts.rag.io_utils import ensure_dir, save_jsonl


ROOT = Path("/home/qingke/DND-AI-BOT")
NORMALIZED_DIR = ROOT / "data/normalized/rules"
CHUNKS_PATH = ROOT / "data/chunks/rules/chunks.jsonl"

WHOLE_DOCUMENT_TYPES = {"class", "race"}
SECTION_HEADING_PATTERN = re.compile(r"^[^\n]{2,60}(?:\s+[A-Za-z][A-Za-z'’ -]{2,})?$")


def build_chunk_id(document_id: str, position: int) -> str:
    return f"{document_id}:{position:04d}"


def chunk_whole_document(document: dict) -> list[ChunkDocument]:
    return [
        ChunkDocument(
            chunk_id=build_chunk_id(document["id"], 1),
            document_id=document["id"],
            knowledge_base=document["knowledge_base"],
            source_type=document["source_type"],
            doc_type=document["doc_type"],
            title=document["title"],
            content=document["content"].strip(),
            language=document["language"],
            section_path=list(document.get("section_path", [])),
            tags=list(document.get("tags", [])),
            aliases=list(document.get("aliases", [])),
            position=1,
            chunk_strategy="whole_document",
        )
    ]


def chunk_rule_sections(document: dict, max_chars: int = 900) -> list[ChunkDocument]:
    chapter_path = list(document.get("section_path", []))
    sections = split_rule_sections(document["content"])
    chunks: list[ChunkDocument] = []
    position = 1

    for section in sections:
        windows = build_section_windows(section["body"], max_chars=max_chars)
        for window in windows:
            content = f"{section['heading']}\n\n{window}".strip()
            chunks.append(
                ChunkDocument(
                    chunk_id=build_chunk_id(document["id"], position),
                    document_id=document["id"],
                    knowledge_base=document["knowledge_base"],
                    source_type=document["source_type"],
                    doc_type=document["doc_type"],
                    title=document["title"],
                    content=content,
                    language=document["language"],
                    section_path=[*chapter_path, section["heading"]],
                    tags=list(document.get("tags", [])),
                    aliases=list(document.get("aliases", [])),
                    position=position,
                    chunk_strategy="section_window",
                )
            )
            position += 1

    if chunks:
        return chunks
    return chunk_whole_document(document)


def chunk_rule_document(document: dict) -> list[ChunkDocument]:
    if document["doc_type"] in WHOLE_DOCUMENT_TYPES:
        return chunk_whole_document(document)
    return chunk_rule_sections(document)


def split_rule_sections(content: str) -> list[dict]:
    lines = [line.strip() for line in content.splitlines()]
    sections: list[dict] = []
    current_heading: str | None = None
    current_lines: list[str] = []

    for line in lines:
        if not line:
            current_lines.append("")
            continue
        if is_section_heading(line):
            if current_heading is not None and joined_text(current_lines):
                sections.append({"heading": current_heading, "body": joined_text(current_lines)})
            current_heading = line
            current_lines = []
            continue
        if current_heading is not None:
            current_lines.append(line)

    if current_heading is not None and joined_text(current_lines):
        sections.append({"heading": current_heading, "body": joined_text(current_lines)})

    return sections


def is_section_heading(line: str) -> bool:
    if line.startswith("第 ") and "章" in line:
        return False
    if line.startswith(("•", "-", "1.", "2.", "3.", "4.", "5.")):
        return False
    if len(line) > 60:
        return False
    if any(mark in line for mark in ("。", "？", "！", "；")):
        return False
    if "：" in line and not line.endswith((")", "）")):
        return False
    return bool(SECTION_HEADING_PATTERN.match(line))


def build_section_windows(body: str, max_chars: int = 900) -> list[str]:
    paragraphs = [paragraph.strip() for paragraph in body.split("\n\n") if paragraph.strip()]
    if not paragraphs:
        return []

    windows: list[str] = []
    current_parts: list[str] = []
    current_length = 0

    for paragraph in paragraphs:
        paragraph_length = len(paragraph)
        if current_parts and current_length + 2 + paragraph_length > max_chars:
            windows.append("\n\n".join(current_parts))
            current_parts = [paragraph]
            current_length = paragraph_length
            continue
        current_parts.append(paragraph)
        current_length = paragraph_length if current_length == 0 else current_length + 2 + paragraph_length

    if current_parts:
        windows.append("\n\n".join(current_parts))

    return windows


def joined_text(lines: list[str]) -> str:
    text = "\n".join(lines)
    text = re.sub(r"\n{3,}", "\n\n", text).strip()
    return text


def chunk_rules(normalized_dir: Path, output_path: Path) -> list[ChunkDocument]:
    ensure_dir(output_path.parent)
    chunks: list[ChunkDocument] = []

    for path in sorted(normalized_dir.glob("*.json")):
        document = json.loads(path.read_text(encoding="utf-8"))
        chunks.extend(chunk_rule_document(document))

    save_jsonl(output_path, [chunk.to_dict() for chunk in chunks])
    return chunks


def main() -> None:
    chunk_rules(NORMALIZED_DIR, CHUNKS_PATH)
    print(CHUNKS_PATH)


if __name__ == "__main__":
    main()
