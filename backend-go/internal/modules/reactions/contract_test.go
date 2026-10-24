package reactions

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/testdb"
)

// reactionsContractState 契约用例上下文：PG 侧 reactions.user_id 有 users 外键，
// 预置 3 个互动用户；目标 ID 无外键约束（moment/comment 行不必真实存在）。
type reactionsContractState struct {
	ctx       context.Context
	userA     string
	userB     string
	userC     string
	momentID  string
	commentID string
}

func TestReactionsRepositoryContract(t *testing.T) {
	userA := "it_reactions_ra"
	userB := "it_reactions_rb"
	userC := "it_reactions_rc"
	momentID := "it_reactions_moment_1"
	commentID := "it_reactions_comment_1"

	impls := []testdb.ContractImpl[Repository, reactionsContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (Repository, reactionsContractState, func()) {
				state := reactionsContractState{
					ctx: context.Background(), userA: userA, userB: userB, userC: userC,
					momentID: momentID, commentID: commentID,
				}
				return NewInMemoryRepository(), state, nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (Repository, reactionsContractState, func()) {
				testdb.SkipIfShort(t)
				testdb.Open(t)
				txFactory, cleanup := testdb.BeginScope(t)
				for _, id := range []string{userA, userB, userC} {
					testdb.InsertUser(t, txFactory, id, "nick_"+id)
				}
				state := reactionsContractState{
					ctx: context.Background(), userA: userA, userB: userB, userC: userC,
					momentID: momentID, commentID: commentID,
				}
				return NewPostgresRepository(txFactory), state, cleanup
			},
		},
	}

	react := func(t *testing.T, repo Repository, state reactionsContractState, targetType ReactTarget, targetID, userID, emoji string, want ReactionOutcome) {
		t.Helper()
		got, err := repo.React(state.ctx, targetType, targetID, userID, emoji)
		if err != nil {
			t.Fatalf("react %s/%s %s: %v", targetType, targetID, emoji, err)
		}
		if got != want {
			t.Fatalf("react outcome = %v, want %v", got, want)
		}
	}

	aggregate := func(t *testing.T, repo Repository, state reactionsContractState, targetType ReactTarget, targetID, viewerID string) []Reaction {
		t.Helper()
		recs, err := repo.Aggregate(state.ctx, targetType, targetID, viewerID)
		if err != nil {
			t.Fatalf("aggregate %s/%s: %v", targetType, targetID, err)
		}
		return recs
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[Repository, reactionsContractState]{
		{
			Name: "react outcome is three-state and aggregate tracks viewer emoji",
			Run: func(t *testing.T, repo Repository, state reactionsContractState) {
				react(t, repo, state, ReactTargetMoment, state.momentID, state.userA, "👍", ReactionInserted)
				react(t, repo, state, ReactTargetMoment, state.momentID, state.userA, "👍", ReactionUnchanged)
				react(t, repo, state, ReactTargetMoment, state.momentID, state.userA, "❤️", ReactionSwitched)

				recs := aggregate(t, repo, state, ReactTargetMoment, state.momentID, state.userA)
				if len(recs) != 1 || recs[0].Emoji != "❤️" || recs[0].Count != 1 || !recs[0].ReactedByMe {
					t.Fatalf("aggregate after switch = %#v", recs)
				}
				anonymous := aggregate(t, repo, state, ReactTargetMoment, state.momentID, "")
				if len(anonymous) != 1 || anonymous[0].ReactedByMe {
					t.Fatalf("anonymous aggregate = %#v", anonymous)
				}

				changed, err := repo.Unreact(state.ctx, ReactTargetMoment, state.momentID, state.userA)
				if err != nil || !changed {
					t.Fatalf("unreact = changed:%v err:%v", changed, err)
				}
				if changed, err := repo.Unreact(state.ctx, ReactTargetMoment, state.momentID, state.userA); err != nil || changed {
					t.Fatalf("unreact again = changed:%v err:%v", changed, err)
				}
				if recs := aggregate(t, repo, state, ReactTargetMoment, state.momentID, state.userA); len(recs) != 0 {
					t.Fatalf("aggregate after unreact = %#v", recs)
				}
				if changed, err := repo.Unreact(state.ctx, ReactTargetComment, state.commentID, state.userB); err != nil || changed {
					t.Fatalf("unreact never-reacted = changed:%v err:%v", changed, err)
				}
			},
		},
		{
			Name: "list reactions is newest-first, filterable and pageable",
			Run: func(t *testing.T, repo Repository, state reactionsContractState) {
				for _, step := range []struct {
					user  string
					emoji string
				}{{state.userA, "👍"}, {state.userB, "❤️"}, {state.userC, "👍"}} {
					react(t, repo, state, ReactTargetMoment, state.momentID, step.user, step.emoji, ReactionInserted)
					time.Sleep(3 * time.Millisecond)
				}
				page1, next, err := repo.ListReactions(state.ctx, ReactTargetMoment, state.momentID, "", 2, "")
				if err != nil {
					t.Fatalf("list page1: %v", err)
				}
				if len(page1) != 2 || next == "" {
					t.Fatalf("page1 = %d items next=%q", len(page1), next)
				}
				if page1[0].User.UserID != state.userC || page1[1].User.UserID != state.userB {
					t.Fatalf("page1 order = %#v", page1)
				}
				for _, item := range page1 {
					if _, err := time.Parse(time.RFC3339Nano, item.ReactedAt); err != nil {
						t.Fatalf("reacted_at must be RFC3339: %q", item.ReactedAt)
					}
					if item.User.UserID == "" {
						t.Fatal("reaction user_id must be non-empty")
					}
				}
				page2, next2, err := repo.ListReactions(state.ctx, ReactTargetMoment, state.momentID, "", 2, next)
				if err != nil {
					t.Fatalf("list page2: %v", err)
				}
				if len(page2) != 1 || next2 != "" || page2[0].User.UserID != state.userA {
					t.Fatalf("page2 = %#v next=%q", page2, next2)
				}
				filtered, _, err := repo.ListReactions(state.ctx, ReactTargetMoment, state.momentID, "👍", 10, "")
				if err != nil {
					t.Fatalf("list filtered: %v", err)
				}
				if len(filtered) != 2 || filtered[0].User.UserID != state.userC || filtered[1].User.UserID != state.userA {
					t.Fatalf("emoji filter = %#v", filtered)
				}
				empty, _, err := repo.ListReactions(state.ctx, ReactTargetMoment, state.momentID, "😢", 10, "")
				if err != nil {
					t.Fatalf("list no-match emoji: %v", err)
				}
				if len(empty) != 0 {
					t.Fatalf("no-match emoji = %#v", empty)
				}
				if _, _, err := repo.ListReactions(state.ctx, ReactTargetMoment, state.momentID, "", 2, "not-a-cursor"); !errors.Is(err, ErrInvalidCursor) {
					t.Fatalf("invalid cursor err = %v, want ErrInvalidCursor", err)
				}
			},
		},
		{
			Name: "switching emoji refreshes reaction recency",
			Run: func(t *testing.T, repo Repository, state reactionsContractState) {
				react(t, repo, state, ReactTargetMoment, state.momentID, state.userA, "👍", ReactionInserted)
				before := state.ctx
				_ = before
				time.Sleep(3 * time.Millisecond)
				react(t, repo, state, ReactTargetMoment, state.momentID, state.userB, "❤️", ReactionInserted)
				time.Sleep(3 * time.Millisecond)
				first, _, err := repo.ListReactions(state.ctx, ReactTargetMoment, state.momentID, "", 10, "")
				if err != nil {
					t.Fatalf("list before switch: %v", err)
				}
				if first[0].User.UserID != state.userB {
					t.Fatalf("before switch order = %#v", first)
				}
				react(t, repo, state, ReactTargetMoment, state.momentID, state.userA, "😂", ReactionSwitched)
				after, _, err := repo.ListReactions(state.ctx, ReactTargetMoment, state.momentID, "", 10, "")
				if err != nil {
					t.Fatalf("list after switch: %v", err)
				}
				if after[0].User.UserID != state.userA || after[0].Emoji != "😂" {
					t.Fatalf("after switch order = %#v", after)
				}
			},
		},
		{
			Name: "aggregate counts, ordering and target isolation",
			Run: func(t *testing.T, repo Repository, state reactionsContractState) {
				react(t, repo, state, ReactTargetMoment, state.momentID, state.userA, "👍", ReactionInserted)
				react(t, repo, state, ReactTargetMoment, state.momentID, state.userB, "👍", ReactionInserted)
				react(t, repo, state, ReactTargetMoment, state.momentID, state.userC, "❤️", ReactionInserted)
				react(t, repo, state, ReactTargetComment, state.commentID, state.userA, "👍", ReactionInserted)

				momentRecs := aggregate(t, repo, state, ReactTargetMoment, state.momentID, state.userB)
				if len(momentRecs) != 2 {
					t.Fatalf("moment aggregate = %#v", momentRecs)
				}
				if momentRecs[0].Emoji != "👍" || momentRecs[0].Count != 2 || !momentRecs[0].ReactedByMe {
					t.Fatalf("moment aggregate[0] = %#v", momentRecs[0])
				}
				if momentRecs[1].Emoji != "❤️" || momentRecs[1].Count != 1 || momentRecs[1].ReactedByMe {
					t.Fatalf("moment aggregate[1] = %#v", momentRecs[1])
				}
				commentRecs := aggregate(t, repo, state, ReactTargetComment, state.commentID, state.userA)
				if len(commentRecs) != 1 || commentRecs[0].Emoji != "👍" || commentRecs[0].Count != 1 || !commentRecs[0].ReactedByMe {
					t.Fatalf("comment aggregate = %#v", commentRecs)
				}
				if empty := aggregate(t, repo, state, ReactTargetMoment, "it_reactions_missing", state.userA); len(empty) != 0 {
					t.Fatalf("missing target aggregate = %#v", empty)
				}
			},
		},
		{
			Name: "aggregate batch matches per-target aggregates",
			Run: func(t *testing.T, repo Repository, state reactionsContractState) {
				react(t, repo, state, ReactTargetMoment, state.momentID, state.userA, "👍", ReactionInserted)
				react(t, repo, state, ReactTargetMoment, state.momentID, state.userB, "❤️", ReactionInserted)
				react(t, repo, state, ReactTargetComment, state.commentID, state.userA, "😂", ReactionInserted)
				batch, err := repo.AggregateBatch(state.ctx, ReactTargetMoment, []string{state.momentID, state.commentID, "it_reactions_missing"}, state.userA)
				if err != nil {
					t.Fatalf("aggregate batch: %v", err)
				}
				if recs := batch[state.momentID]; len(recs) != 2 {
					t.Fatalf("batch moment = %#v", recs)
				}
				if recs := batch[state.commentID]; len(recs) != 0 {
					t.Fatalf("batch comment (other target type) = %#v", recs)
				}
				if recs := batch["it_reactions_missing"]; len(recs) != 0 {
					t.Fatalf("batch missing = %#v", recs)
				}
				emptyBatch, err := repo.AggregateBatch(state.ctx, ReactTargetMoment, nil, state.userA)
				if err != nil || len(emptyBatch) != 0 {
					t.Fatalf("empty batch = %#v err=%v", emptyBatch, err)
				}
			},
		},
		{
			Name: "list reacted targets is newest-first, pageable and type-scoped",
			Run: func(t *testing.T, repo Repository, state reactionsContractState) {
				targetA := "it_reactions_target_a"
				targetB := "it_reactions_target_b"
				targetC := "it_reactions_target_c"
				for _, target := range []string{targetA, targetB, targetC} {
					react(t, repo, state, ReactTargetMoment, target, state.userA, "👍", ReactionInserted)
					time.Sleep(3 * time.Millisecond)
				}
				react(t, repo, state, ReactTargetComment, state.commentID, state.userA, "👍", ReactionInserted)

				page1, next, err := repo.ListReactedTargets(state.ctx, ReactTargetMoment, state.userA, 2, "")
				if err != nil {
					t.Fatalf("targets page1: %v", err)
				}
				if len(page1) != 2 || next == "" || page1[0] != targetC || page1[1] != targetB {
					t.Fatalf("targets page1 = %#v next=%q", page1, next)
				}
				page2, next2, err := repo.ListReactedTargets(state.ctx, ReactTargetMoment, state.userA, 2, next)
				if err != nil {
					t.Fatalf("targets page2: %v", err)
				}
				if len(page2) != 1 || next2 != "" || page2[0] != targetA {
					t.Fatalf("targets page2 = %#v next=%q", page2, next2)
				}
				others, _, err := repo.ListReactedTargets(state.ctx, ReactTargetMoment, state.userB, 10, "")
				if err != nil {
					t.Fatalf("targets other user: %v", err)
				}
				if len(others) != 0 {
					t.Fatalf("other user targets = %#v", others)
				}
				if _, _, err := repo.ListReactedTargets(state.ctx, ReactTargetMoment, state.userA, 2, "not-a-cursor"); !errors.Is(err, ErrInvalidCursor) {
					t.Fatalf("invalid target cursor err = %v, want ErrInvalidCursor", err)
				}
			},
		},
	})
}
