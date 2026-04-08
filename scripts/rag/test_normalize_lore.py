import json
import tempfile
import unittest
from pathlib import Path

from scripts.rag.normalize_lore import (
    build_lore_document_id,
    normalize_lore,
    split_markdown_sections,
    strip_lore_noise,
)


class NormalizeLoreTests(unittest.TestCase):
    def test_strip_lore_noise_removes_images_and_credit_sections(self) -> None:
        markdown = (
            "# 城市\n\n"
            "![image](https://example.com/a.png)\n\n"
            "# 图片鸣谢\n\n"
            "封面作者\n\n"
            "# 死寂的世界\n\n"
            "这里只有城市。\n"
        )

        cleaned = strip_lore_noise(markdown)

        self.assertNotIn("![image]", cleaned)
        self.assertNotIn("图片鸣谢", cleaned)
        self.assertIn("# 死寂的世界", cleaned)

    def test_split_markdown_sections_returns_top_level_sections(self) -> None:
        markdown = (
            "# 死寂的世界\n\n"
            "这里只有城市。\n\n"
            "# 凝固的天空\n\n"
            "太阳静止不动。\n"
        )

        sections = split_markdown_sections(markdown)

        self.assertEqual([section["title"] for section in sections], ["死寂的世界", "凝固的天空"])
        self.assertIn("这里只有城市", sections[0]["content"])

    def test_build_lore_document_id(self) -> None:
        self.assertEqual(build_lore_document_id("default-setting", "死寂的世界"), "lore:default-setting:dead-world")
        self.assertEqual(build_lore_document_id("default-setting", "凝固的天空"), "lore:default-setting:frozen-sky")

    def test_normalize_lore_writes_section_documents(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            markdown_path = root / "settings.md"
            output_dir = root / "normalized"
            markdown_path.write_text(
                "# 死寂的世界\n\n这里只有城市。\n\n# 凝固的天空\n\n太阳静止不动。\n",
                encoding="utf-8",
            )

            documents = normalize_lore(markdown_path, output_dir, setting_id="default-setting")

            self.assertEqual(len(documents), 2)
            dead_world_path = output_dir / "lore-default-setting-dead-world.json"
            self.assertTrue(dead_world_path.exists())

            dead_world = json.loads(dead_world_path.read_text(encoding="utf-8"))
            self.assertEqual(dead_world["knowledge_base"], "lore")
            self.assertEqual(dead_world["doc_type"], "setting_section")
            self.assertEqual(dead_world["title"], "死寂的世界")
            self.assertEqual(dead_world["section_path"], ["死寂的世界"])
            self.assertIn("这里只有城市", dead_world["content"])


if __name__ == "__main__":
    unittest.main()
