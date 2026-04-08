from __future__ import annotations

from dataclasses import asdict, dataclass, field


@dataclass
class NormalizedDocument:
    id: str
    knowledge_base: str
    source_type: str
    source_file: str
    title: str
    doc_type: str
    language: str
    content: str
    chapter: str = ""
    setting_id: str = ""
    book: str = ""
    section_path: list[str] = field(default_factory=list)
    tags: list[str] = field(default_factory=list)
    aliases: list[str] = field(default_factory=list)

    def to_dict(self) -> dict:
        return asdict(self)


@dataclass
class ChunkDocument:
    chunk_id: str
    document_id: str
    knowledge_base: str
    source_type: str
    doc_type: str
    title: str
    content: str
    language: str
    section_path: list[str] = field(default_factory=list)
    tags: list[str] = field(default_factory=list)
    aliases: list[str] = field(default_factory=list)
    position: int = 1
    chunk_strategy: str = ""

    def to_dict(self) -> dict:
        return asdict(self)
