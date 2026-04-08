import json
import tempfile
import unittest
from pathlib import Path

from scripts.rag.chunk_lore import (
    build_chunk_id,
    build_paragraph_windows,
    chunk_lore,
    chunk_lore_document,
    split_paragraphs,
)


class ChunkLoreTests(unittest.TestCase):
    def test_build_chunk_id(self) -> None:
        self.assertEqual(build_chunk_id("lore:default-setting:dead-world", 1), "lore:default-setting:dead-world:0001")

    def test_split_paragraphs_returns_non_empty_paragraphs(self) -> None:
        content = "第一段。\n\n第二段。\n\n\n第三段。"

        paragraphs = split_paragraphs(content)

        self.assertEqual(paragraphs, ["第一段。", "第二段。", "第三段。"])

    def test_build_paragraph_windows_respects_max_chars(self) -> None:
        paragraphs = ["第一段内容。" * 10, "第二段内容。" * 10, "第三段内容。" * 10]

        windows = build_paragraph_windows(paragraphs, max_chars=60, overlap_chars=20)

        self.assertGreaterEqual(len(windows), 2)
        self.assertTrue(all(len(window) <= 80 for window in windows))

    def test_chunk_lore_document_builds_paragraph_window_chunks(self) -> None:
        document = {
            "id": "lore:default-setting:dead-world",
            "knowledge_base": "lore",
            "source_type": "background_md",
            "doc_type": "setting_section",
            "title": "死寂的世界",
            "content": "第一段。\n\n第二段。" + ("很长的内容" * 60) + "\n\n第三段。",
            "language": "zh",
            "section_path": ["死寂的世界"],
            "tags": ["setting_section", "dead-world"],
            "aliases": ["死寂的世界"],
        }

        chunks = chunk_lore_document(document, max_chars=120, overlap_chars=30)

        self.assertGreaterEqual(len(chunks), 2)
        self.assertTrue(all(chunk.chunk_strategy == "paragraph_window" for chunk in chunks))
        self.assertEqual(chunks[0].section_path, ["死寂的世界"])
        self.assertEqual(chunks[0].title, "死寂的世界")

    def test_chunk_lore_writes_jsonl(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            normalized_dir = root / "normalized"
            output_path = root / "chunks" / "chunks.jsonl"
            normalized_dir.mkdir(parents=True, exist_ok=True)

            document = {
                "id": "lore:default-setting:dead-world",
                "knowledge_base": "lore",
                "source_type": "background_md",
                "source_file": "settings.md",
                "title": "死寂的世界",
                "doc_type": "setting_section",
                "language": "zh",
                "content": "第一段。\n\n第二段。\n\n第三段。",
                "chapter": "",
                "setting_id": "default-setting",
                "book": "",
                "section_path": ["死寂的世界"],
                "tags": ["setting_section", "dead-world"],
                "aliases": ["死寂的世界"],
            }
            (normalized_dir / "dead-world.json").write_text(json.dumps(document, ensure_ascii=False), encoding="utf-8")

            chunks = chunk_lore(normalized_dir, output_path)

            self.assertGreaterEqual(len(chunks), 1)
            self.assertTrue(output_path.exists())
            rows = [json.loads(line) for line in output_path.read_text(encoding="utf-8").splitlines() if line.strip()]
            self.assertEqual(len(rows), len(chunks))
            self.assertEqual(rows[0]["document_id"], "lore:default-setting:dead-world")


if __name__ == "__main__":
    unittest.main()
