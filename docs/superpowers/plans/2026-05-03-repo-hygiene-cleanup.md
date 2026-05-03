# Repo Hygiene Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove tracked build cache and local binaries, tighten ignore rules, refresh README, and add repository metadata for GitHub About/topics.

**Architecture:** Keep source layout unchanged. Only clean repository hygiene artifacts and documentation. Treat GitHub metadata as declarative repo config under `.github/settings.yml`.

**Tech Stack:** Git, Go repository conventions, Markdown, YAML.

---

### Task 1: Remove tracked local build artifacts

**Files:**
- Modify: Git index entries under `.cache/go-build/**`
- Delete: `.cache/go-build/`
- Delete: `main`

- [ ] Remove `.cache/go-build` from the git index.
- [ ] Delete local `.cache/go-build` directory.
- [ ] Delete local root binary `main`.

### Task 2: Tighten ignore rules and repo metadata

**Files:**
- Modify: `.gitignore`
- Create: `.github/settings.yml`

- [ ] Add ignore rules for Go build cache, local binaries, reports, and editor noise without widening scope unnecessarily.
- [ ] Add GitHub repository description and topics in `.github/settings.yml`.

### Task 3: Refresh README

**Files:**
- Modify: `README.md`

- [ ] Remove stale or chatty wording.
- [ ] Clarify project scope as backend service.
- [ ] Add concise local run / build hygiene guidance.
- [ ] Align README wording with repository About/topics metadata.

### Task 4: Verify and commit

**Files:**
- Verify: repository status / diff

- [ ] Run `git diff --check`.
- [ ] Run `git status --short`.
- [ ] Commit with `update: Clean repository artifacts and metadata`.
