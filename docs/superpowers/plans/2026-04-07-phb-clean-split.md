# PHB Clean And Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Clean the extracted PHB text and split it into chapter files plus high-confidence race/class entry files under `data/raw/Rules`.

**Architecture:** Add two focused Python scripts under `scripts/rules`: one cleans the extracted PHB text into a normalized `cleaned.txt`, and one splits the cleaned text into chapter and entry text files. Use `unittest` regression tests to lock cleaning and splitting behavior before implementation.

**Tech Stack:** Python 3 standard library, `unittest`, filesystem outputs under `data/raw/Rules`

---

### Task 1: Add Cleaner Tests

**Files:**
- Create: `scripts/rules/test_clean_phb_text.py`
- Test: `scripts/rules/test_clean_phb_text.py`

- [ ] **Step 1: Write the failing test**

```python
def test_clean_text_removes_page_markers_and_control_chars(self):
    raw = "===== PAGE 1 =====\n中译\x01\x01v1.6 版\n第 1 章 冒险\n"
    cleaned = clean_text(raw)
    self.assertNotIn("===== PAGE 1 =====", cleaned)
    self.assertNotIn("\x01", cleaned)
    self.assertIn("第 1 章 冒险", cleaned)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python3 -m unittest scripts.rules.test_clean_phb_text -v`
Expected: FAIL because `clean_text` does not exist yet

- [ ] **Step 3: Write minimal implementation**

```python
def clean_text(raw: str) -> str:
    ...
```

- [ ] **Step 4: Run test to verify it passes**

Run: `python3 -m unittest scripts.rules.test_clean_phb_text -v`
Expected: PASS

### Task 2: Add Splitter Tests

**Files:**
- Create: `scripts/rules/test_split_phb_text.py`
- Test: `scripts/rules/test_split_phb_text.py`

- [ ] **Step 1: Write the failing test**

```python
def test_splitter_extracts_chapters_and_race_class_entries(self):
    cleaned = (
        "第 2 章 种族\n"
        "矮人\n矮人的正文。\n"
        "精灵\n精灵的正文。\n"
        "第 3 章 职业\n"
        "战士\n战士的正文。\n"
    )
    result = split_cleaned_text(cleaned)
    self.assertIn("chapter-02-种族.txt", result["chapters"])
    self.assertIn("race-精灵.txt", result["entries"])
    self.assertIn("class-战士.txt", result["entries"])
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python3 -m unittest scripts.rules.test_split_phb_text -v`
Expected: FAIL because `split_cleaned_text` does not exist yet

- [ ] **Step 3: Write minimal implementation**

```python
def split_cleaned_text(cleaned: str) -> dict[str, dict[str, str]]:
    ...
```

- [ ] **Step 4: Run test to verify it passes**

Run: `python3 -m unittest scripts.rules.test_split_phb_text -v`
Expected: PASS

### Task 3: Implement Cleaner Script

**Files:**
- Create: `scripts/rules/clean_phb_text.py`
- Modify: `scripts/rules/test_clean_phb_text.py`

- [ ] **Step 1: Implement `clean_text` and `main`**

```python
def clean_text(raw: str) -> str:
    ...

def main() -> None:
    ...
```

- [ ] **Step 2: Run targeted cleaner tests**

Run: `python3 -m unittest scripts.rules.test_clean_phb_text -v`
Expected: PASS

- [ ] **Step 3: Run cleaner against real input**

Run: `python3 scripts/rules/clean_phb_text.py`
Expected: writes `data/raw/Rules/DnD玩家手册5ePHBv1.6版_cleaned.txt`

### Task 4: Implement Splitter Script

**Files:**
- Create: `scripts/rules/split_phb_text.py`
- Modify: `scripts/rules/test_split_phb_text.py`

- [ ] **Step 1: Implement `split_cleaned_text` and `main`**

```python
def split_cleaned_text(cleaned: str) -> dict[str, dict[str, str]]:
    ...

def main() -> None:
    ...
```

- [ ] **Step 2: Run targeted splitter tests**

Run: `python3 -m unittest scripts.rules.test_split_phb_text -v`
Expected: PASS

- [ ] **Step 3: Run splitter against real cleaned input**

Run: `python3 scripts/rules/split_phb_text.py`
Expected: writes chapter files to `data/raw/Rules/split/chapters/` and race/class files to `data/raw/Rules/split/entries/`

### Task 5: Verify Real Outputs

**Files:**
- Inspect: `data/raw/Rules/DnD玩家手册5ePHBv1.6版_cleaned.txt`
- Inspect: `data/raw/Rules/split/chapters/`
- Inspect: `data/raw/Rules/split/entries/`

- [ ] **Step 1: Run the full test suite for the new scripts**

Run: `python3 -m unittest scripts.rules.test_clean_phb_text scripts.rules.test_split_phb_text -v`
Expected: PASS

- [ ] **Step 2: Inspect generated files**

Run: `ls -lh data/raw/Rules/split/chapters data/raw/Rules/split/entries`
Expected: chapter files plus race/class entry files exist

- [ ] **Step 3: Sample-check output content**

Run: `sed -n '1,80p' data/raw/Rules/split/chapters/chapter-02-种族.txt`
Expected: cleaned chapter text without page markers

