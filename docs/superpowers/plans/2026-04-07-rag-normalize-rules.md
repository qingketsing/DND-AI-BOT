# RAG Normalize Rules Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert the cleaned and split PHB rule text files into standardized normalized JSON documents under `data/normalized/rules`.

**Architecture:** Add shared dataclass-style document models and JSON IO helpers under `scripts/rag`, then implement a focused `normalize_rules.py` that reads `data/raw/Rules/split/chapters` and `data/raw/Rules/split/entries` and emits one normalized JSON per source file.

**Tech Stack:** Python 3 standard library, `dataclasses`, `json`, `unittest`

---

### Task 1: Add Failing Tests For Rules Normalization

**Files:**
- Create: `scripts/rag/test_normalize_rules.py`
- Test: `scripts/rag/test_normalize_rules.py`

- [ ] **Step 1: Write the failing test**

```python
def test_normalize_rules_builds_class_document(tmp_path):
    ...
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python3 -m unittest /home/qingke/DND-AI-BOT/scripts/rag/test_normalize_rules.py -v`
Expected: FAIL because normalization code does not exist yet

- [ ] **Step 3: Implement minimal code to pass**

```python
def normalize_rules(...):
    ...
```

- [ ] **Step 4: Run test to verify it passes**

Run: `python3 -m unittest /home/qingke/DND-AI-BOT/scripts/rag/test_normalize_rules.py -v`
Expected: PASS

### Task 2: Add Shared Models And IO Helpers

**Files:**
- Create: `scripts/rag/common_models.py`
- Create: `scripts/rag/io_utils.py`
- Modify: `scripts/rag/test_normalize_rules.py`

- [ ] **Step 1: Add dataclass models**

```python
@dataclass
class NormalizedDocument:
    ...
```

- [ ] **Step 2: Add JSON IO helpers**

```python
def save_json(path: Path, data: dict) -> None:
    ...
```

- [ ] **Step 3: Re-run normalization tests**

Run: `python3 -m unittest /home/qingke/DND-AI-BOT/scripts/rag/test_normalize_rules.py -v`
Expected: PASS

### Task 3: Implement Rules Normalization

**Files:**
- Create: `scripts/rag/normalize_rules.py`
- Modify: `scripts/rag/test_normalize_rules.py`

- [ ] **Step 1: Implement rules normalization entry points**

```python
def normalize_rules(cleaned_path: Path, chapters_dir: Path, entries_dir: Path, output_dir: Path) -> list[NormalizedDocument]:
    ...
```

- [ ] **Step 2: Implement chapter and entry document builders**

```python
def build_chapter_documents(chapters_dir: Path) -> list[NormalizedDocument]:
    ...

def build_entry_documents(entries_dir: Path) -> list[NormalizedDocument]:
    ...
```

- [ ] **Step 3: Run tests**

Run: `python3 -m unittest /home/qingke/DND-AI-BOT/scripts/rag/test_normalize_rules.py -v`
Expected: PASS

### Task 4: Generate Real Normalized Outputs

**Files:**
- Inspect: `data/normalized/rules/`

- [ ] **Step 1: Run normalization on real PHB split files**

Run: `python3 /home/qingke/DND-AI-BOT/scripts/rag/normalize_rules.py`
Expected: normalized JSON files written to `data/normalized/rules`

- [ ] **Step 2: Verify output count and sample files**

Run: `find /home/qingke/DND-AI-BOT/data/normalized/rules -maxdepth 1 -type f | wc -l`
Expected: one JSON per chapter/entry source file

- [ ] **Step 3: Sample-check a class and chapter document**

Run: `sed -n '1,120p' /home/qingke/DND-AI-BOT/data/normalized/rules/rules-phb-class-wizard.json`
Expected: JSON with `knowledge_base`, `doc_type`, `title`, `section_path`, and full `content`
