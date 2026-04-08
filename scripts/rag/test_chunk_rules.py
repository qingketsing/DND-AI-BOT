import json
import tempfile
import unittest
from pathlib import Path

from scripts.rag.chunk_rules import (
    build_chunk_id,
    chunk_rule_document,
    chunk_rule_sections,
    chunk_rules,
)


class ChunkRulesTests(unittest.TestCase):
    def test_build_chunk_id(self) -> None:
        self.assertEqual(build_chunk_id("rules:phb:class:wizard", 1), "rules:phb:class:wizard:0001")

    def test_chunk_rule_document_keeps_whole_document_for_class(self) -> None:
        document = {
            "id": "rules:phb:class:wizard",
            "knowledge_base": "rules",
            "source_type": "phb",
            "doc_type": "class",
            "title": "法师",
            "content": "法师 Wizard\n\n法师是奥术施法者。",
            "language": "zh",
            "section_path": ["第 3 章：职业", "法师"],
            "tags": ["class", "wizard"],
            "aliases": ["法师", "Wizard"],
        }

        chunks = chunk_rule_document(document)

        self.assertEqual(len(chunks), 1)
        self.assertEqual(chunks[0].chunk_strategy, "whole_document")
        self.assertEqual(chunks[0].title, "法师")
        self.assertIn("法师是奥术施法者", chunks[0].content)

    def test_chunk_rule_sections_splits_chapter_by_subsections(self) -> None:
        document = {
            "id": "rules:phb:chapter:09-combat",
            "knowledge_base": "rules",
            "source_type": "phb",
            "doc_type": "chapter",
            "title": "战斗",
            "content": (
                "第 9 章：战斗 Combat\n\n"
                "战斗流程 The Order of Combat\n"
                "第一段。\n\n"
                "先攻 Initiative\n"
                "第二段。\n\n"
                "反应 Reactions\n"
                "第三段。"
            ),
            "language": "zh",
            "section_path": ["第 9 章：战斗"],
            "tags": ["chapter", "战斗"],
            "aliases": ["战斗"],
        }

        chunks = chunk_rule_sections(document, max_chars=40)

        self.assertGreaterEqual(len(chunks), 3)
        self.assertTrue(all(chunk.chunk_strategy == "section_window" for chunk in chunks))
        self.assertEqual(chunks[0].section_path, ["第 9 章：战斗", "战斗流程 The Order of Combat"])
        self.assertIn("第一段", chunks[0].content)

    def test_chunk_rules_writes_jsonl(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            normalized_dir = root / "normalized"
            output_path = root / "chunks" / "chunks.jsonl"
            normalized_dir.mkdir(parents=True, exist_ok=True)

            chapter_document = {
                "id": "rules:phb:chapter:09-combat",
                "knowledge_base": "rules",
                "source_type": "phb",
                "source_file": "chapter-09-战斗.txt",
                "title": "战斗",
                "doc_type": "chapter",
                "language": "zh",
                "content": "第 9 章：战斗 Combat\n\n战斗流程 The Order of Combat\n第一段。",
                "chapter": "第 9 章：战斗",
                "book": "PHB",
                "section_path": ["第 9 章：战斗"],
                "tags": ["chapter", "战斗"],
                "aliases": ["战斗"],
            }
            class_document = {
                "id": "rules:phb:class:wizard",
                "knowledge_base": "rules",
                "source_type": "phb",
                "source_file": "class-法师.txt",
                "title": "法师",
                "doc_type": "class",
                "language": "zh",
                "content": "法师 Wizard\n\n法师是奥术施法者。",
                "chapter": "第 3 章：职业",
                "book": "PHB",
                "section_path": ["第 3 章：职业", "法师"],
                "tags": ["class", "wizard"],
                "aliases": ["法师", "Wizard"],
            }
            (normalized_dir / "chapter.json").write_text(json.dumps(chapter_document, ensure_ascii=False), encoding="utf-8")
            (normalized_dir / "class.json").write_text(json.dumps(class_document, ensure_ascii=False), encoding="utf-8")

            chunks = chunk_rules(normalized_dir, output_path)

            self.assertGreaterEqual(len(chunks), 2)
            self.assertTrue(output_path.exists())
            rows = [json.loads(line) for line in output_path.read_text(encoding="utf-8").splitlines() if line.strip()]
            self.assertEqual(len(rows), len(chunks))
            self.assertEqual(rows[-1]["document_id"], "rules:phb:class:wizard")


if __name__ == "__main__":
    unittest.main()
