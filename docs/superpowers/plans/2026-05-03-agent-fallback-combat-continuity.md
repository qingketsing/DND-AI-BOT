# Agent Fallback and Combat Continuity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce visible long-session failures by making fallback replies action-aware and by preloading structured encounter state into every agent turn.

**Architecture:** Keep the existing runtime and tool chain intact. Fix the bug at the orchestration layer: preload encounter summary alongside recent messages/game state/session memory, and make fallback replies depend on inferred user intent so combat and scene actions do not falsely appear resolved or reset.

**Tech Stack:** Go, existing agent runtime, encounter service, session context prompt, Go tests.

---

### Task 1: Lock regression behavior with tests

**Files:**
- Create: `internal/service/agent_fallback_test.go`
- Modify: `internal/app/context_preload_test.go`
- Modify: `internal/agent/prompt/session_context_test.go`

- [ ] **Step 1: Write failing fallback tests**

Add tests covering:
- combat-action fallback returns "action not resolved / turn not advanced" style wording
- status-query fallback returns "cannot reliably read structured state" wording
- exploration-action fallback returns "scene not advanced" wording

- [ ] **Step 2: Run fallback tests to verify RED**

Run: `GOCACHE=/tmp/go-build go test ./internal/service -run 'TestDefaultAgentFallbackResponder'`
Expected: FAIL because current fallback is generic.

- [ ] **Step 3: Write failing preloaded-context tests**

Extend preloaded context tests so they require encounter summary content:
- round / turn index
- current combatant
- visible combatants with hp / ac / side

- [ ] **Step 4: Run preloaded-context tests to verify RED**

Run: `GOCACHE=/tmp/go-build go test ./internal/app ./internal/agent/prompt -run 'TestBuildPreloadedContextPrompt|TestComposePreloadedSessionContextPrompt'`
Expected: FAIL because encounter is not loaded/rendered yet.

### Task 2: Implement minimal orchestration fix

**Files:**
- Modify: `internal/service/agent_fallback.go`
- Modify: `internal/app/context_preload.go`
- Modify: `internal/agent/prompt/session_context.go`
- Modify: `internal/app/app.go`

- [ ] **Step 1: Implement action-aware fallback**

Use existing intent classifier to branch fallback copy for:
- combat action
- status query / session recall
- exploration action
- default generic case

- [ ] **Step 2: Implement encounter preload**

Load encounter in `buildPreloadedContextPrompt`, ignoring `ErrEncounterNotFound` the same way game state missing is ignored.

- [ ] **Step 3: Render compact encounter summary**

Add a prompt renderer that emits:
- encounter id / round / turn index
- current actor
- visible combatants in stable order with `name`, `side`, `hp`, `ac`, `status`

- [ ] **Step 4: Wire encounter reader into app runtime runner**

Pass `EncounterService` into `newRuntimeAgentRunner` via a small reader interface so each turn can preload encounter state.

### Task 3: Verification

**Files:**
- Test: `internal/service/agent_fallback_test.go`
- Test: `internal/app/context_preload_test.go`
- Test: `internal/agent/prompt/session_context_test.go`

- [ ] **Step 1: Run targeted tests**

Run:
`GOCACHE=/tmp/go-build go test ./internal/service ./internal/app ./internal/agent/prompt`

Expected: PASS

- [ ] **Step 2: Run broader agent regression tests**

Run:
`GOCACHE=/tmp/go-build go test ./internal/agent/... ./internal/app/... ./internal/service/...`

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/plans/2026-05-03-agent-fallback-combat-continuity.md internal/service/agent_fallback.go internal/service/agent_fallback_test.go internal/app/context_preload.go internal/app/context_preload_test.go internal/app/app.go internal/agent/prompt/session_context.go internal/agent/prompt/session_context_test.go
git commit -m "update: Improve fallback and combat context continuity"
```
