from __future__ import annotations

import json
import sys
from pathlib import Path

if __package__ is None or __package__ == "":
    sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from scripts.rag.common_models import ChunkDocument
from scripts.rag.io_utils import ensure_dir, save_jsonl


ROOT = Path("/home/qingke/DND-AI-BOT")
NORMALIZED_DIR = ROOT / "data/normalized/lore"
CHUNKS_PATH = ROOT / "data/chunks/lore/chunks.jsonl"


def build_chunk_id(document_id: str, position: int) -> str:
    return f"{document_id}:{position:04d}"


def split_paragraphs(content: str) -> list[str]:
    return [paragraph.strip() for paragraph in content.split("\n\n") if paragraph.strip()]


def build_paragraph_windows(
    paragraphs: list[str],
    max_chars: int,
    overlap_chars: int,
) -> list[str]:
    if not paragraphs:
        return []

    windows: list[str] = []
    index = 0

    while index < len(paragraphs):
        current_parts: list[str] = []
        current_length = 0
        end_index = index

        while end_index < len(paragraphs):
            paragraph = paragraphs[end_index]
            addition = len(paragraph) if not current_parts else len(paragraph) + 2
            if current_parts and current_length + addition > max_chars:
                break
            current_parts.append(paragraph)
            current_length += addition
            end_index += 1

        if not current_parts:
            current_parts.append(paragraphs[index])
            end_index = index + 1

        windows.append("\n\n".join(current_parts))

        if end_index >= len(paragraphs):
            break

        overlap_start = end_index
        overlap_length = 0
        while overlap_start > index:
            candidate = paragraphs[overlap_start - 1]
            candidate_length = len(candidate) if overlap_length == 0 else len(candidate) + 2
            if overlap_length + candidate_length > overlap_chars:
                break
            overlap_start -= 1
            overlap_length += candidate_length

        if overlap_start == end_index:
            index = end_index
        else:
            index = max(overlap_start, index + 1)

    return windows


def chunk_lore_document(
    document: dict,
    max_chars: int = 800,
    overlap_chars: int = 120,
) -> list[ChunkDocument]:
    paragraphs = split_paragraphs(document["content"])
    windows = build_paragraph_windows(paragraphs, max_chars=max_chars, overlap_chars=overlap_chars)
    chunks: list[ChunkDocument] = []

    for position, window in enumerate(windows, start=1):
        chunks.append(
            ChunkDocument(
                chunk_id=build_chunk_id(document["id"], position),
                document_id=document["id"],
                knowledge_base=document["knowledge_base"],
                source_type=document["source_type"],
                doc_type=document["doc_type"],
                title=document["title"],
                content=window,
                language=document["language"],
                section_path=list(document.get("section_path", [])),
                tags=list(document.get("tags", [])),
                aliases=list(document.get("aliases", [])),
                position=position,
                chunk_strategy="paragraph_window",
            )
        )

    return chunks


def chunk_lore(normalized_dir: Path, output_path: Path) -> list[ChunkDocument]:
    ensure_dir(output_path.parent)
    chunks: list[ChunkDocument] = []

    for path in sorted(normalized_dir.glob("*.json")):
        document = json.loads(path.read_text(encoding="utf-8"))
        chunks.extend(chunk_lore_document(document))

    save_jsonl(output_path, [chunk.to_dict() for chunk in chunks])
    return chunks


def main() -> None:
    chunk_lore(NORMALIZED_DIR, CHUNKS_PATH)
    print(CHUNKS_PATH)


if __name__ == "__main__":
    main()
