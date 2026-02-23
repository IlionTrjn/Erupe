# Erupe Technical Debt & Suggested Next Steps

> Last updated: 2026-02-22

This document tracks actionable technical debt items discovered during a codebase audit. It complements `anti-patterns.md` (which covers structural patterns) by focusing on specific, fixable items with file paths and line numbers.

## Table of Contents

- [High Priority](#high-priority)
  - [1. Broken game features (gameplay-impacting TODOs)](#1-broken-game-features-gameplay-impacting-todos)
  - [2. Test gaps on critical paths](#2-test-gaps-on-critical-paths)
- [Medium Priority](#medium-priority)
  - [3. Logging anti-patterns](#3-logging-anti-patterns)
- [Low Priority](#low-priority)
  - [4. CI updates](#4-ci-updates)
- [Completed Items](#completed-items)
- [Suggested Execution Order](#suggested-execution-order)

---

## High Priority

### 1. Broken game features (gameplay-impacting TODOs)

These TODOs represent features that are visibly broken for players.

| Location | Issue | Impact |
|----------|-------|--------|
| `model_character.go:88,101,113` | `TODO: fix bookshelf data pointer` for G10-ZZ, F4-F5, and S6 versions | Wrong pointer corrupts character save reads for three game versions. Offset analysis shows all three are off by exactly 14810 vs the consistent delta pattern of other fields — but needs validation against actual save data. |
| `handlers_achievement.go:125` | `TODO: Notify on rank increase` — always returns `false` | Achievement rank-up notifications are silently suppressed. Requires understanding what `MhfDisplayedAchievement` (currently an empty handler) sends to track "last displayed" state. |
| `handlers_guild_info.go:443` | `TODO: Enable GuildAlliance applications` — hardcoded `true` | Guild alliance applications are always open regardless of setting. Needs research into where the toggle originates. |
| `handlers_session.go:394` | `TODO(Andoryuuta): log key index off-by-one` | Known off-by-one in log key indexing is unresolved |
| `handlers_session.go:535` | `TODO: This case might be <=G2` | Uncertain version detection in switch case |
| `handlers_session.go:698` | `TODO: Retail returned the number of clients in quests` | Player count reported to clients does not match retail behavior |

### 2. Test gaps on critical paths

**Handler files with no test file (6 remaining):**

| File | Priority | Reason |
|------|----------|--------|
| `handlers_data_paper.go` | MEDIUM | Daily paper data |
| `handlers_seibattle.go` | MEDIUM | Sei battle system |
| `handlers_scenario.go` | LOW | Mostly complete, uses repo |
| `handlers_distitem.go` | LOW | Distribution items |
| `handlers_guild_mission.go` | LOW | Guild missions |
| `handlers_kouryou.go` | LOW | Kouryou system |

**Repository files with no store-level test file (17 total):**

`repo_achievement.go`, `repo_cafe.go`, `repo_distribution.go`, `repo_diva.go`, `repo_festa.go`, `repo_gacha.go`, `repo_goocoo.go`, `repo_house.go`, `repo_mail.go`, `repo_mercenary.go`, `repo_misc.go`, `repo_rengoku.go`, `repo_scenario.go`, `repo_session.go`, `repo_shop.go`, `repo_stamp.go`, `repo_tower.go`

These are validated indirectly through mock-based handler tests but have no SQL-level integration tests.

---

## Medium Priority

### 3. Logging anti-patterns

~~**a) `fmt.Sprintf` inside structured logger calls (6 sites):**~~ **Fixed.** All 6 sites now use `zap.Uint32`/`zap.Uint8`/`zap.String` structured fields instead of `fmt.Sprintf`.

~~**b) 20+ silently discarded SJIS encoding errors in packet parsing:**~~ **Fixed.** All call sites now use `SJISToUTF8Lossy()` which logs decode errors at `slog.Debug` level.

---

## Low Priority

### 4. CI updates

- ~~`codecov-action@v4` could be updated to `v5` (current stable)~~ **Fixed.** Updated to `codecov-action@v5`.
- No coverage threshold is enforced — coverage is uploaded but regressions aren't caught

---

## Completed Items

Items resolved since the original audit:

| # | Item | Resolution |
|---|------|------------|
| ~~3~~ | **Sign server has no repository layer** | Fully refactored with `repo_interfaces.go`, `repo_user.go`, `repo_session.go`, `repo_character.go`, and mock tests. All 8 previously-discarded error paths are now handled. |
| ~~4~~ | **Split `repo_guild.go`** | Split from 1004 lines into domain-focused files: `repo_guild.go` (466 lines, core CRUD), `repo_guild_posts.go`, `repo_guild_alliance.go`, `repo_guild_adventure.go`, `repo_guild_hunt.go`, `repo_guild_cooking.go`, `repo_guild_rp.go`. |
| ~~6~~ | **Inconsistent transaction API** | All call sites now use `BeginTxx(context.Background(), nil)` with deferred rollback. |
| ~~7~~ | **`LoopDelay` config has no Viper default** | `viper.SetDefault("LoopDelay", 50)` added in `config/config.go`. |
| — | **Monthly guild item claim** (`handlers_guild.go:389`) | Now tracks per-character per-type monthly claims via `stamps` table. |
| — | **Handler test coverage (4 files)** | Tests added for `handlers_session.go`, `handlers_gacha.go`, `handlers_plate.go`, `handlers_shop.go`. |
| — | **Handler test coverage (`handlers_commands.go`)** | 62 tests covering all 12 commands, disabled-command gating, op overrides, error paths, raviente with semaphore, course enable/disable/locked, reload with players/objects. |
| — | **Entrance server raw SQL** | Refactored to repository interfaces (`repo_interfaces.go`, `repo_session.go`, `repo_server.go`). |
| — | **Guild daily RP rollover** (`handlers_guild_ops.go:148`) | Implemented via lazy rollover in `handlers_guild.go:110-119` using `RolloverDailyRP()`. Stale TODO removed. |
| — | **Typos** (`sys_session.go`, `handlers_session.go`) | "For Debuging" and "offical" typos already fixed in previous commits. |
| — | **`db != nil` guard** (`handlers_session.go:322`) | Investigated — this guard is intentional. Test servers run without repos; the guard protects the entire logout path from nil repo dereferences. Not a leaky abstraction. |

---

## Suggested Execution Order

Based on remaining impact:

1. ~~**Add tests for `handlers_commands.go`**~~ — **Done.** 62 tests covering all 12 commands (ban, timer, PSN, reload, key quest, rights, course, raviente, teleport, discord, playtime, help), disabled-command gating, op overrides, error paths, and `initCommands`.
2. **Fix bookshelf data pointer** (`model_character.go`) — corrupts saves for three game versions (needs save data validation)
3. **Fix achievement rank-up notifications** (`handlers_achievement.go:125`) — needs protocol research on `MhfDisplayedAchievement`
4. **Add coverage threshold** to CI — prevents regressions
