# RAG Normalize Lore Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert `data/raw/Background/settings.md` into normalized lore JSON documents under `data/normalized/lore`.

**Architecture:** Add a focused `normalize_lore.py` script that strips image/publishing noise, splits the markdown by top-level headings, and emits one normalized lore document per content section. Reuse the shared dataclass and JSON helpers from `scripts/rag`.

**Tech Stack:** Python 3 standard library, `re`, `json`, `unittest`

---

### Task 1: Add Failing Tests For Lore Normalization

**Files:**
- Create: `scripts/rag/test_normalize_lore.py`
- Test: `scripts/rag/test_normalize_lore.py`

- [ ] **Step 1: Write the failing test**

```python
def test_normalize_lore_splits_content_sections(tmp_path):
    ...
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python3 -m unittest /home/qingke/DND-AI-BOT/scripts/rag/test_normalize_lore.py -v`
Expected: FAIL because lore normalization code does not exist yet

- [ ] **Step 3: Implement minimal code to pass**

```python
def normalize_lore(...):
    ...
```

- [ ] **Step 4: Run test to verify it passes**

Run: `python3 -m unittest /home/qingke/DND-AI-BOT/scripts/rag/test_normalize_lore.py -v`
Expected: PASS

### Task 2: Implement Lore Normalization

**Files:**
- Create: `scripts/rag/normalize_lore.py`
- Modify: `scripts/rag/test_normalize_lore.py`

- [ ] **Step 1: Implement markdown cleanup and section splitting**

```python
def strip_lore_noise(markdown_text: str) -> str:
    ...

def split_markdown_sections(markdown_text: str) -> list[dict]:
    ...
```

- [ ] **Step 2: Implement normalized lore builders**

```python
def build_lore_documents(sections: list[dict], source_file: str, setting_id: str) -> list[NormalizedDocument]:
    ...
```

- [ ] **Step 3: Run tests**

Run: `python3 -m unittest /home/qingke/DND-AI-BOT/scripts/rag/test_normalize_lore.py -v`
Expected: PASS

### Task 3: Generate Real Lore Outputs

**Files:**
- Inspect: `data/normalized/lore/`

- [ ] **Step 1: Run normalization on the real settings markdown**

Run: `python3 /home/qingke/DND-AI-BOT/scripts/rag/normalize_lore.py`
Expected: normalized JSON files written to `data/normalized/lore`

- [ ] **Step 2: Verify output count**

Run: `find /home/qingke/DND-AI-BOT/data/normalized/lore -maxdepth 1 -type f | wc -l`
Expected: one JSON per content section

- [ ] **Step 3: Sample-check a generated lore file**

Run: `sed -n '1,160p' /home/qingke/DND-AI-BOT/data/normalized/lore/lore-default-setting-dead-world.json`
Expected: JSON with `knowledge_base`, `doc_type`, `title`, `section_path`, and cleaned section `content`
