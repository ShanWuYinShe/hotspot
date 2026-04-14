# Claude Workflow Governance Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reorganize HotSpot's top-level Claude workflow guidance so low-risk tasks stay fast while high-risk tasks still follow stronger process and verification rules.

**Architecture:** Keep the root repository guidance short and layered, move detailed orchestration into the workflow skill, and update README text so humans and agents see the same workflow model.

**Tech Stack:** Markdown documentation, repository skill metadata, git worktrees.

---

### Task 1: Rewrite root workflow guidance

**Files:**
- Modify: `CLAUDE.md`
- Test: `README.md` (consistency review)

**Step 1: Write the target structure in the document**

Add sections for repository map, hard constraints, task levels, verification levels, and workflow entrypoints.

**Step 2: Remove duplicated or overly rigid process wording**

Keep only the global rules that should apply to all tasks. Move heavy orchestration details out of the root file.

**Step 3: Add the `.worktrees/` default and finish-stage history guidance**

State that all writes happen in worktrees and that the single-commit requirement applies before syncing back to `main`.

**Step 4: Review for clarity and overlap**

Make sure the root guidance does not restate subproject-level build and test commands.

### Task 2: Update repository README workflow guidance

**Files:**
- Modify: `README.md`
- Test: `CLAUDE.md` (consistency review)

**Step 1: Update the collaboration section**

Explain the new layered model: hard constraints, default workflow, and risk-based skill escalation.

**Step 2: Document task levels and validation levels briefly**

Keep this concise and link back to `CLAUDE.md` for authoritative behavior.

**Step 3: Document `.worktrees/` as the default isolated workspace location**

Keep this visible in the common entrypoints section.

### Task 3: Adjust the workflow skill entrypoint

**Files:**
- Modify: `.claude/skills/hotspot-workflow/SKILL.md`
- Modify: `.claude/skills/hotspot-workflow/references/feature/implementation.md`
- Modify: `.claude/skills/hotspot-workflow/references/feature/testing.md`
- Modify: `.claude/skills/hotspot-workflow/references/feature/finish.md`

**Step 1: Update the skill to route by task level and task type**

Clarify that low-risk feature work can continue directly after context review, while broader changes escalate to planning or design.

**Step 2: Remove stop conditions that conflict with the new collaboration model**

Specifically avoid halting just because unrelated uncommitted changes exist outside the task path.

**Step 3: Align testing and finish references with the two-level verification model**

Make local checks the default completion gate and isolated Docker verification the merge gate for integration-sensitive work.

**Step 4: Review for consistency with root guidance**

Check that the skill references match the updated `CLAUDE.md` and `README.md` wording.

### Task 4: Verify the documentation set

**Files:**
- Test: `CLAUDE.md`
- Test: `README.md`
- Test: `.claude/skills/hotspot-workflow/SKILL.md`

**Step 1: Read the updated files together**

Confirm they agree on `.worktrees/`, task levels, verification levels, and workflow entrypoints.

**Step 2: Run a lightweight repository status check**

Run: `git status --short`
Expected: only the intended workflow-governance files are modified.

**Step 3: Commit when requested**

Run: `git add <updated files> && git commit -m "feat(claude): streamline workflow governance"`
Expected: one commit on the worktree branch.
