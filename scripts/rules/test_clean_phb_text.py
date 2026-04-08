import unittest

from scripts.rules.clean_phb_text import clean_text


class CleanPhbTextTests(unittest.TestCase):
    def test_clean_text_removes_page_markers_control_chars_and_front_matter(self) -> None:
        raw = (
            "===== PAGE 1 =====\n"
            "PLAYER'S HANDBOOK\n"
            "第 2 章：种族\n"
            "矮人 By 幻 C\n"
            "第 1 章：一步步创建角色 Step-by-Step Characters\n"
            "中译\x01\x01v1.6 版\n"
            "真正的正文段落。\n"
        )

        cleaned = clean_text(raw)

        self.assertNotIn("===== PAGE 1 =====", cleaned)
        self.assertNotIn("\x01", cleaned)
        self.assertNotIn("矮人 By 幻 C", cleaned)
        self.assertIn("第 1 章：一步步创建角色 Step-by-Step Characters", cleaned)
        self.assertIn("真正的正文段落。", cleaned)

    def test_clean_text_collapses_excess_blank_lines(self) -> None:
        raw = (
            "第 1 章：一步步创建角色 Step-by-Step Characters\n\n\n"
            "第一段。\n\n\n\n第二段。\n"
        )

        cleaned = clean_text(raw)

        self.assertNotIn("\n\n\n", cleaned)
        self.assertIn("第一段。\n\n第二段。", cleaned)

    def test_clean_text_removes_toc_page_numbers_and_part_markers(self) -> None:
        raw = (
            "目录 Contents\n"
            "前言 Preface\n"
            "索引 Index\n\n"
            "前言 Preface\n"
            "前言正文。\n"
            "4\n"
            "第 2 部分\n"
            "正文继续。\n"
        )

        cleaned = clean_text(raw)

        self.assertTrue(cleaned.startswith("前言 Preface"))
        self.assertNotIn("索引 Index", cleaned)
        self.assertNotIn("\n4\n", cleaned)
        self.assertNotIn("第 2 部分", cleaned)

    def test_clean_text_repairs_split_chinese_words(self) -> None:
        raw = (
            "前言 Preface\n"
            "灰 鹰 世 界 里 的 自 由 城 邦\n"
            "半 精 灵 和 法 师 一 起 行 动。\n"
        )

        cleaned = clean_text(raw)

        self.assertIn("灰鹰世界里的自由城邦", cleaned)
        self.assertIn("半精灵和法师一起行动。", cleaned)

    def test_clean_text_reflows_wrapped_paragraph_lines_but_keeps_headings(self) -> None:
        raw = (
            "前言 Preface\n"
            "第一行断在这里\n"
            "第二行接在后面。\n\n"
            "法师 Wizard\n"
            "法师第一行\n"
            "法师第二行。\n\n"
            "奥术学者 Scholars of the Arcane\n"
            "说明第一行\n"
            "说明第二行。\n"
        )

        cleaned = clean_text(raw)

        self.assertIn("前言 Preface\n第一行断在这里第二行接在后面。", cleaned)
        self.assertIn("法师 Wizard\n法师第一行法师第二行。", cleaned)
        self.assertIn("\n\n奥术学者 Scholars of the Arcane\n说明第一行说明第二行。", cleaned)


if __name__ == "__main__":
    unittest.main()
