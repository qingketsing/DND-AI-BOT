# README Restore Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore the original non-configuration README content while keeping the corrected configuration and deployment sections.

**Architecture:** Treat the README as a merged document rather than a rewrite. Keep the original narrative sections, file tree, and contributors block, and preserve the newer environment, deployment, and repository hygiene guidance where it improves accuracy.

**Tech Stack:** Markdown, Git

---

### Task 1: Merge README content conservatively

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Restore original narrative sections**

Bring back the original non-configuration sections:
- `什么是DND Game Bot?`
- `如何工作？`
- `游戏规则，回复和当前工作重心`
- `文档和社交媒体`
- `关键特征`
- `文件树介绍`
- `社区贡献者`

- [ ] **Step 2: Keep corrected configuration and deployment sections**

Retain the current updated sections for:
- environment variables
- hybrid retrieval / embedding config
- production security
- latency log controls
- rate limiting
- PostgreSQL backup
- reverse proxy / HTTPS

- [ ] **Step 3: Preserve repository cleanup guidance**

Keep the repository hygiene guidance that documents ignored local artifacts and generated outputs.

- [ ] **Step 4: Validate formatting**

Run:

```bash
git diff --check
```

Expected: no whitespace or merge-format issues.

- [ ] **Step 5: Commit**

```bash
git add README.md docs/superpowers/plans/2026-05-03-readme-restore-plan.md
git commit -m "update: Restore README content and contributors"
```
