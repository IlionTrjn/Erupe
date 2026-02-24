# Erupe Improvement Plan

> Analysis date: 2026-02-24

Actionable improvements identified during a codebase audit. Items are ordered by priority and designed to be tackled sequentially. Complements `anti-patterns.md` and `technical-debt.md`.

## Table of Contents

- [1. Fix swallowed errors with nil-dereference risk](#1-fix-swallowed-errors-with-nil-dereference-risk)
- [2. Fix bookshelf data pointer for three game versions](#2-fix-bookshelf-data-pointer-for-three-game-versions)
- [3. Add error feedback to parseChatCommand](#3-add-error-feedback-to-parsechatcommand)
- [4. Reconcile service layer docs vs reality](#4-reconcile-service-layer-docs-vs-reality)
- [5. Consolidate GuildRepo mocks](#5-consolidate-guildrepo-mocks)
- [6. Add mocks for 8 unmocked repo interfaces](#6-add-mocks-for-8-unmocked-repo-interfaces)
- [7. Extract inline data tables from handler functions](#7-extract-inline-data-tables-from-handler-functions)

---

## 1. Fix swallowed errors with nil-dereference risk

**Priority:** High — latent panics triggered by any DB hiccup.

~30 sites use `_, _` to discard repo/service errors. Three are dangerous because the returned value is used without a nil guard:

| Location | Risk |
|----------|------|
| `handlers_guild_adventure.go:24,48,73` | `guild, _ := guildRepo.GetByCharID(...)` — no nil guard, will panic on DB error |
| `handlers_gacha.go:56` | `fp, gp, gt, _ := userRepo.GetGachaPoints(...)` — balance silently becomes 0, enabling invalid transactions |
| `handlers_house.go:167` | 7 return values from `GetHouseContents`, error discarded entirely |

Additional sites that don't panic but produce silently wrong data:

| Location | Issue |
|----------|-------|
| `handlers_distitem.go:35,111,129` | `distRepo.List()`/`GetItems()` errors become empty results, no logging |
| `handlers_guild_ops.go:30,49` | `guildService.Disband()`/`Leave()` errors swallowed (nil-safe due to `result != nil` guard, but invisible failures) |
| `handlers_shop.go:125,131` | Gacha type/weight lookups discarded |
| `handlers_discord.go:34` | `bcrypt.GenerateFromPassword` error swallowed (only fails on OOM) |

**Fix:** Add error checks with logging and appropriate fail ACKs. For the three high-risk sites, add nil guards at minimum.

**Status:** **Done.** All swallowed errors fixed across 7 files:

- `handlers_guild_adventure.go` — 3 `GetByCharID` calls now check error and nil, early-return with ACK
- `handlers_gacha.go` — `GetGachaPoints` now checks error and returns zeroed response; `GetStepupStatus` logs error
- `handlers_house.go` — `GetHouseContents` now checks error and sends fail ACK; `HasApplication` moved inside nil guard to prevent nil dereference on `ownGuild`; `GetMission` and `GetWarehouseNames` now log errors
- `handlers_distitem.go` — 3 `distRepo` calls now log errors
- `handlers_guild_ops.go` — `Disband` and `Leave` service errors now logged
- `handlers_shop.go` — `GetShopType`, `GetWeightDivisor`, `GetFpointExchangeList` now log errors
- `handlers_discord.go` — `bcrypt.GenerateFromPassword` error now returns early with user-facing message

---

## 2. Fix bookshelf data pointer for three game versions

**Priority:** High — corrupts character save reads.

From `technical-debt.md`: `model_character.go:88,101,113` has `TODO: fix bookshelf data pointer` for G10-ZZ, F4-F5, and S6 versions. All three offsets are off by exactly 14810 vs the consistent delta pattern of other fields. Needs validation against actual save data.

**Fix:** Analyze save data from affected game versions to determine correct offsets. Apply fix and add regression test.

**Status:** Pending.

---

## 3. Add error feedback to parseChatCommand

**Priority:** Medium — improves operator experience with low effort.

`handlers_commands.go:71` is a 351-line switch statement dispatching 12 chat commands. Argument parsing errors (`strconv`, `hex.DecodeString`) are silently swallowed at lines 240, 256, 368, 369. Malformed commands silently use zero values instead of giving the operator feedback.

**Fix:** On parse error, send a chat message back to the player explaining the expected format, then return early. Each command's branch already has access to the session for sending messages.

**Status:** **Done.** All 4 sites now validate parse results and send the existing i18n error messages:

- `hex.DecodeString` (KeyQuest set) — sends kqf.set.error on invalid hex
- `strconv.Atoi` (Rights) — sends rights.error on non-integer
- `strconv.ParseInt` x/y (Teleport) — sends teleport.error on non-integer coords

---

## 4. Reconcile service layer docs vs reality

**Priority:** Medium — documentation mismatch causes confusion for contributors.

The CLAUDE.md architecture section shows a clean `handlers → svc_*.go → repo_*.go` layering, but in practice:

- **GuildService** has 7 methods. **GuildRepo** has 68. Handlers call `guildRepo` directly ~60+ times across 7 guild handler files.
- The 4 services (`GuildService`, `MailService`, `AchievementService`, `GachaService`) were extracted for operations requiring cross-repo coordination (e.g., disband triggers mail), but the majority of handler logic goes directly to repos.

This isn't necessarily wrong — the services exist for multi-repo coordination, not as a mandatory pass-through.

**Fix:** Update the architecture diagram in `CLAUDE.md` to reflect the actual pattern: services are used for cross-repo coordination, handlers call repos directly for simple CRUD. Remove the implication that all handlers go through services. Alternatively, expand service coverage to match the documented architecture, but that is a much larger effort with diminishing returns.

**Status:** Pending.

---

## 5. Consolidate GuildRepo mocks

**Priority:** Low — reduces friction for guild test authoring.

`repo_mocks_test.go` (1004 lines) has two separate GuildRepo mock types:

- `mockGuildRepoForMail` (67 methods, 104 lines) — used by mail tests
- `mockGuildRepoOps` (38 methods, 266 lines) — used by ops/scout tests, with configurable behavior via struct fields

The `GuildRepo` interface has 68 methods. Neither mock implements the full interface. Adding any new `GuildRepo` method requires updating both mocks or compilation fails.

**Fix:** Merge into a single `mockGuildRepo` with all 68 methods as no-op defaults. Use struct fields (as `mockGuildRepoOps` already does for ~15 methods) for configurable returns in tests that need specific behavior.

**Status:** Pending.

---

## 6. Add mocks for 8 unmocked repo interfaces

**Priority:** Low — enables isolated handler tests for more subsystems.

8 of the 21 repo interfaces have no mock implementation: `TowerRepo`, `FestaRepo`, `RengokuRepo`, `DivaRepo`, `EventRepo`, `MiscRepo`, `MercenaryRepo`, `CafeRepo`.

Tests for those handlers either use stub handlers that skip repos or rely on integration tests. This limits the ability to write isolated unit tests.

**Fix:** Add no-op mock implementations for each, following the pattern established by existing mocks.

**Status:** Pending.

---

## 7. Extract inline data tables from handler functions

**Priority:** Low — improves readability.

`handlers_items.go:18` — `handleMsgMhfEnumeratePrice` (164 lines) embeds two large `var` data blocks inline in the function body. These are static data tables, not logic.

**Fix:** Extract to package-level `var` declarations or a dedicated data file (following the pattern of `handlers_data_paper_tables.go`).

**Status:** Pending.
