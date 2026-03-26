---
name: moonhub-change-docs-sync
description: Updates MoonHub package-level docs under pkg/*/docs/, the repository documentation index docs/README.md, CHANGELOG.md, and bilingual README Features (README.md and README_CN.md) after code changes. Use when the user completes a MoonHub feature or fix and asks to sync documentation, refresh the doc index, update the changelog, or keep English and Chinese readmes aligned.
---

# MoonHub change → documentation sync

Run this workflow **after** substantive code changes so docs stay consistent with the canonical patterns in [`pkg/learning/docs/`](../../../pkg/learning/docs/).

## Preconditions

- Identify **which packages** and **subsystems** changed (from diff or user summary).
- Treat **MoonHub repository root** as the working root for all paths below.

## 1. Package-level documentation (`pkg/<package>/docs/`)

Follow the **learning** package as the baseline (structure and tone may vary by subsystem):

| Reference file | Role |
| --- | --- |
| [`pkg/learning/docs/README.md`](../../../pkg/learning/docs/README.md) | Subsystem overview, core components, link to **Documentation Index** at repo [`docs/README.md`](../../../docs/README.md), config pointer |
| [`pkg/learning/docs/CONFIG.md`](../../../pkg/learning/docs/CONFIG.md) | `config.json` keys, defaults, option tables |
| [`pkg/learning/docs/I18N.md`](../../../pkg/learning/docs/I18N.md) | Language / locale behavior (when applicable) |
| [`pkg/learning/docs/EXAMPLES.md`](../../../pkg/learning/docs/EXAMPLES.md), [`SupportedPatterns.md`](../../../pkg/learning/docs/SupportedPatterns.md) | Examples and tables (when applicable) |

**Rules**

- Create or update `pkg/<package>/docs/` for packages that gained public API, config surface, or operator-visible behavior.
- **README.md** in `docs/`: scope, main entry points (files/symbols), integration touchpoints, and a **relative** link to the repo index (same pattern as learning: from `pkg/foo/docs/README.md` use `../../../docs/README.md`).
- **CONFIG.md**: document every new or changed key; keep JSON examples copy-pasteable.
- If the subsystem already has **implementation status** in `docs/implementation/<name>-status.md`, cross-link from package docs and update that status file when behavior or checklist changes.
- Larger subsystems may also use **ARCHITECTURE.md**, **INTEGRATION.md**, **FEATURES.md**, **METRICS.md** (see [`pkg/routing/docs/`](../../../pkg/routing/docs/) or [`pkg/shield/docs/`](../../../pkg/shield/docs/))—add only what matches the change scope.
- **pkg README**: If the package root has `pkg/<package>/README.md`, keep it aligned with `docs/` (short overview + link into `docs/`).

## 2. Repository documentation index (`docs/README.md`)

[`docs/README.md`](../../../docs/README.md) is the **single entry point** for subsystem reading order.

For each affected subsystem, update **all** that apply:

- **Recommended Reading Order (First Time)** — new optional flows (env flags, major subsystems).
- **Documentation Flow by Subsystem** — table rows: ordered doc links (`README` → `CONFIG` → extras) and **Status** row pointing to `docs/implementation/*-status.md`.
- **`docs/implementation/` Overview** — add or rename rows when status files are added/renamed.
- **Package Structure Overview** — new `pkg/` rows or corrected descriptions when packages are added or repurposed.
- **Cross-links** — e.g. [`web/README.md`](../../../web/README.md), [`CLAUDE.md`](../../../CLAUDE.md) if the change affects web or contributor-facing architecture (match depth of existing entries).

Keep table ordering and link style consistent with adjacent sections.

## 3. Changelog (`CHANGELOG.md`)

- Prepend a **new top-level dated section** (newest first), matching the style of existing entries: **Summary**, **New Features** (with `` `pkg/...` `` paths), **Files Changed**, and **Documentation** / **Technical Details** when useful.
- Do not remove prior history; append or insert only.

## 4. Root README Features — English (`README.md`)

- Under **## Features → ### Implemented**, add or adjust **one bullet per user-visible capability**, mirroring existing bullets: short title, em dash, one-line description, **relative links** to `docs/implementation/…` and/or `pkg/…/docs/…` where appropriate.
- If something moves from planned to implemented, update **### Planned** and the **Completed (click to expand)** `<details>` block per existing conventions.

## 5. Root README — Chinese (`README_CN.md`)

- Keep **structure and link targets** aligned with `README.md`: **## 功能特性 → ### 已实现** / **### 计划中** / **已完成** details.
- **Wording**: translate the English feature bullets faithfully; preserve the same link paths.
- **Do not** add or maintain a long **更新日志** section in `README_CN.md`; release notes live only in [`CHANGELOG.md`](../../../CHANGELOG.md). The CN README may keep a **single line** linking to `CHANGELOG.md` (same idea as the English README).

## 6. Verification checklist

Before finishing:

- [ ] Every changed package with config or public behavior has updated `pkg/…/docs/` (and status doc if the subsystem uses one).
- [ ] `docs/README.md` tables and reading order reflect the new or changed subsystem.
- [ ] `CHANGELOG.md` has a new dated section describing the change.
- [ ] `README.md` **Implemented** (and Planned/details if needed) matches shipped behavior.
- [ ] `README_CN.md` matches `README.md` for Features (no duplicate changelog body; pointer to `CHANGELOG.md` only).
- [ ] All new links are **relative** and resolve from the repo root layout.

## Additional resources

- Templates for changelog and feature bullets: [reference.md](reference.md)
