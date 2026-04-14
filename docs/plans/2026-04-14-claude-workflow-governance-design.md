# Claude Workflow Governance Design

## Context

HotSpot already prefers documentation-guided collaboration, skill entrypoints, and a small number of strict scripts. The current workflow guidance is directionally correct, but several rules are expressed at the top level as if they apply to every task. That makes simple requests slower than necessary and creates tension between direct execution and mandatory process steps.

## Goals

- Keep Claude Code aligned with HotSpot's repository boundaries and delivery standards.
- Preserve a fast path for local, low-risk work.
- Move heavy process steps to explicit risk-based triggers.
- Keep one clear entrypoint for worktree setup, verification, and finish steps.

## Chosen Approach

### 1. Split repository guidance into three layers

- Hard constraints: global rules that must always hold.
- Default workflow: the normal path for common tasks.
- On-demand skills: specialized flows triggered only when task type or risk requires them.

This keeps the root guidance short and stable while letting workflow skills stay richer and more opinionated.

### 2. Add task levels with a default fast path

- L0: read-only analysis or questions.
- L1: local low-risk changes in one subproject with limited surface area.
- L2: medium changes needing a short execution plan.
- L3: feature, architecture, contract, or cross-surface changes needing design first.

L0 and L1 should execute directly after context review. L2 and L3 can escalate into plan or design flows.

### 3. Split verification into development-complete and merge-ready

- Development complete: relevant local checks, targeted tests, no debug leftovers.
- Ready to merge: development-complete checks plus isolated Docker verification when the work touches integration boundaries.

This preserves quality gates without making every small task pay the full integration cost.

### 4. Shorten top-level guidance and push orchestration down

- Root `CLAUDE.md` should define boundaries, task levels, and default entrypoints.
- Subproject `CLAUDE.md` files continue to own technical constraints and local commands.
- `README.md` explains the workflow to both humans and agents.
- `hotspot-workflow` remains the repository workflow entrypoint, but should route by task risk instead of forcing every task through the same heavy path.

## Workflow Implications

- Default worktree directory becomes `.worktrees/`.
- The repository keeps the single-commit-on-main goal, but this is enforced as a finish requirement instead of being phrased as an implementation-time limitation.
- Isolated verification remains required for DB, migration, cross-service, or deployment-sensitive work.

## Expected Outcome

After this change, Claude Code should move faster on small tasks, remain predictable on medium tasks, and still respect HotSpot's stronger merge and verification standards on higher-risk work.
