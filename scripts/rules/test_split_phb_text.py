import unittest

from scripts.rules.split_phb_text import split_cleaned_text


class SplitPhbTextTests(unittest.TestCase):
    def test_splitter_extracts_chapters_and_race_class_entries(self) -> None:
        cleaned = (
            "第 2 章：种族 Races\n"
            "矮人 Dwarf\n"
            "矮人的正文。\n\n"
            "精灵 Elf\n"
            "精灵的正文。\n\n"
            "第 3 章：职业 Classes\n"
            "战士 Fighter\n"
            "战士的正文。\n\n"
            "法师 Wizard\n"
            "法师的正文。\n"
        )

        result = split_cleaned_text(cleaned)

        self.assertIn("chapter-02-种族.txt", result["chapters"])
        self.assertIn("chapter-03-职业.txt", result["chapters"])
        self.assertIn("race-精灵.txt", result["entries"])
        self.assertIn("class-战士.txt", result["entries"])
        self.assertIn("法师的正文。", result["entries"]["class-法师.txt"])

    def test_splitter_keeps_entry_within_its_source_chapter(self) -> None:
        cleaned = (
            "第 2 章：种族 Races\n"
            "矮人 Dwarf\n"
            "矮人的正文。\n\n"
            "第 3 章：职业 Classes\n"
            "战士 Fighter\n"
            "战士的正文。\n"
        )

        result = split_cleaned_text(cleaned)

        self.assertNotIn("战士的正文。", result["entries"]["race-矮人.txt"])
        self.assertIn("战士的正文。", result["entries"]["class-战士.txt"])

    def test_splitter_only_uses_heading_lines_as_entry_markers(self) -> None:
        cleaned = (
            "第 2 章：种族 Races\n"
            "精灵 Elf\n"
            "真正的精灵条目正文。\n"
            "精灵则会选择在人口密集的社会中谋生。\n"
        )

        result = split_cleaned_text(cleaned)

        self.assertEqual(
            result["entries"]["race-精灵.txt"],
            "精灵 Elf\n真正的精灵条目正文。\n精灵则会选择在人口密集的社会中谋生。\n",
        )

    def test_splitter_does_not_treat_chinese_only_subheading_as_main_class_entry(self) -> None:
        cleaned = (
            "第 3 章：职业 Classes\n"
            "战士 Fighter\n"
            "真正的战士条目正文。\n"
            "战士\n"
            "一个子小节。\n"
        )

        result = split_cleaned_text(cleaned)

        self.assertEqual(
            result["entries"]["class-战士.txt"],
            "战士 Fighter\n真正的战士条目正文。\n战士\n一个子小节。\n",
        )


if __name__ == "__main__":
    unittest.main()
