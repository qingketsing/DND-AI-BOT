import json
import tempfile
import unittest
from pathlib import Path

from scripts.rag.normalize_rules import (
    build_rule_document_id,
    infer_rule_doc_type,
    normalize_rules,
)


class NormalizeRulesTests(unittest.TestCase):
    def test_infer_rule_doc_type_from_file_name(self) -> None:
        self.assertEqual(infer_rule_doc_type("class-法师.txt"), "class")
        self.assertEqual(infer_rule_doc_type("race-精灵.txt"), "race")
        self.assertEqual(infer_rule_doc_type("chapter-02-种族.txt"), "chapter")

    def test_build_rule_document_id(self) -> None:
        self.assertEqual(build_rule_document_id("class-法师.txt"), "rules:phb:class:wizard")
        self.assertEqual(build_rule_document_id("race-精灵.txt"), "rules:phb:race:elf")
        self.assertEqual(build_rule_document_id("chapter-02-种族.txt"), "rules:phb:chapter:02-races")

    def test_normalize_rules_writes_class_and_chapter_documents(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            cleaned_path = root / "cleaned.txt"
            chapters_dir = root / "chapters"
            entries_dir = root / "entries"
            output_dir = root / "normalized"
            chapters_dir.mkdir()
            entries_dir.mkdir()

            cleaned_path.write_text("cleaned", encoding="utf-8")
            (chapters_dir / "chapter-02-种族.txt").write_text("第 2 章：种族\n正文。", encoding="utf-8")
            (entries_dir / "class-法师.txt").write_text("法师 Wizard\n法师正文。", encoding="utf-8")

            documents = normalize_rules(cleaned_path, chapters_dir, entries_dir, output_dir)

            self.assertEqual(len(documents), 2)

            wizard_path = output_dir / "rules-phb-class-wizard.json"
            chapter_path = output_dir / "rules-phb-chapter-02-races.json"
            self.assertTrue(wizard_path.exists())
            self.assertTrue(chapter_path.exists())

            wizard = json.loads(wizard_path.read_text(encoding="utf-8"))
            self.assertEqual(wizard["knowledge_base"], "rules")
            self.assertEqual(wizard["doc_type"], "class")
            self.assertEqual(wizard["title"], "法师")
            self.assertEqual(wizard["section_path"], ["第 3 章：职业", "法师"])
            self.assertIn("法师正文", wizard["content"])

            chapter = json.loads(chapter_path.read_text(encoding="utf-8"))
            self.assertEqual(chapter["doc_type"], "chapter")
            self.assertEqual(chapter["title"], "种族")
            self.assertEqual(chapter["chapter"], "第 2 章：种族")


if __name__ == "__main__":
    unittest.main()
