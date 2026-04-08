from __future__ import annotations

import re
from pathlib import Path


RAW_PATH = Path("/home/qingke/DND-AI-BOT/data/raw/Rules/DnD玩家手册5ePHBv1.6版_extracted.txt")
CLEANED_PATH = Path("/home/qingke/DND-AI-BOT/data/raw/Rules/DnD玩家手册5ePHBv1.6版_cleaned.txt")
FIRST_CHAPTER_MARKER = "第 1 章：一步步创建角色 Step-by-Step Characters"
PREFACE_MARKER = "前言 Preface"


def strip_control_characters(text: str) -> str:
    """删除除换行和制表符外的控制字符。"""
    return "".join(ch for ch in text if ch in "\n\t" or ord(ch) >= 32)


def normalize_whitespace(text: str) -> str:
    """统一空白字符并压缩多余空行。"""
    text = text.replace("\r\n", "\n").replace("\r", "\n").replace("\t", " ")
    text = re.sub(r"[ \u3000]+", " ", text)
    text = re.sub(r" *\n *", "\n", text)
    text = re.sub(r"\n{3,}", "\n\n", text)
    return text.strip() + "\n"


def repair_split_chinese_words(text: str) -> str:
    """修复 PDF 提取造成的中文断词空格。"""
    return re.sub(r"(?<=[\u4e00-\u9fff]) (?=[\u4e00-\u9fff])", "", text)


def is_heading_line(line: str) -> bool:
    """判断一行是否应保留为独立标题。"""
    stripped = line.strip()
    if not stripped:
        return False
    if stripped.startswith("•"):
        return False
    if re.match(r"^(前言|简介|第\s*[0-9一二三四五六七八九十]+\s*部分|第\s*\d+\s*章|第\s*[一二三四五六七八九十]+\s*章|附录\s*[A-Z])", stripped):
        return True
    if any(mark in stripped for mark in "，。！？；"):
        return False
    if len(stripped) <= 80 and re.fullmatch(
        r"[\u4e00-\u9fff0-9（）()《》：:·'’“”\- ]{1,40}\s+[A-Za-z][A-Za-z0-9 &'’().:/-]{1,40}",
        stripped,
    ):
        return True
    return False


def reflow_paragraph_lines(text: str) -> str:
    """将连续正文行合并为自然段，保留标题、空行和列表项。"""
    output: list[str] = []
    paragraph: list[str] = []

    def flush_paragraph() -> None:
        if paragraph:
            output.append("".join(paragraph))
            paragraph.clear()

    for raw_line in text.splitlines():
        line = raw_line.strip()
        if not line:
            flush_paragraph()
            if output and output[-1] != "":
                output.append("")
            continue
        if is_heading_line(line) or line.startswith("•"):
            flush_paragraph()
            output.append(line)
            continue
        paragraph.append(line)

    flush_paragraph()
    return "\n".join(output)


def clean_text(raw: str) -> str:
    """清洗提取文本，保留从正文第一章开始的主体内容。"""
    text = strip_control_characters(raw)
    text = re.sub(r"^===== PAGE \d+ =====\n?", "", text, flags=re.MULTILINE)
    preface_indices = [match.start() for match in re.finditer(re.escape(PREFACE_MARKER), text)]
    marker_index = preface_indices[-1] if preface_indices else -1
    if marker_index == -1:
        marker_index = text.find(FIRST_CHAPTER_MARKER)
    if marker_index != -1:
        text = text[marker_index:]
    text = re.sub(r"^.*\bBy\b.*$", "", text, flags=re.MULTILINE)
    text = re.sub(r"^PLAYER.?S HANDBOOK.*$", "", text, flags=re.MULTILINE)
    text = re.sub(r"^龙与地下城.*$", "", text, flags=re.MULTILINE)
    text = re.sub(r"^\s*第\s*[0-9一二三四五六七八九十]+\s*部分\s*$", "", text, flags=re.MULTILINE)
    text = re.sub(r"^\s*[0-9]{1,3}\s*$", "", text, flags=re.MULTILINE)
    text = normalize_whitespace(text)
    text = repair_split_chinese_words(text)
    text = reflow_paragraph_lines(text)
    text = normalize_whitespace(text)
    return text


def main() -> None:
    raw = RAW_PATH.read_text(encoding="utf-8")
    cleaned = clean_text(raw)
    CLEANED_PATH.write_text(cleaned, encoding="utf-8")
    print(CLEANED_PATH)


if __name__ == "__main__":
    main()
