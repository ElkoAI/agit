package app

import (
	"fmt"
	"time"

	"github.com/agentgit/agit/internal/db"
)

type PruneResult struct {
	Total      int
	Kept       int
	Deleted    int
	DeletedIDs []int64
}

func PruneCheckpoints(ctx *Context, keepLast int, keepDays int, dryRun bool) (PruneResult, error) {
	if keepLast <= 0 && keepDays <= 0 {
		return PruneResult{}, fmt.Errorf("keep-last or keep-days must be set")
	}

	cps, err := db.ListCheckpoints(ctx.DB, ctx.WorkspaceID, 0)
	if err != nil {
		return PruneResult{}, err
	}

	keep := map[int64]bool{}
	if keepLast > 0 {
		for i := 0; i < keepLast && i < len(cps); i++ {
			keep[cps[i].ID] = true
		}
	}

	var cutoff time.Time
	if keepDays > 0 {
		cutoff = time.Now().Add(-time.Duration(keepDays) * 24 * time.Hour)
	}

	toDelete := make([]int64, 0)
	for _, cp := range cps {
		if keep[cp.ID] {
			continue
		}
		if keepDays > 0 {
			if cp.CreatedAt.After(cutoff) {
				continue
			}
		}
		toDelete = append(toDelete, cp.ID)
	}

	res := PruneResult{
		Total:      len(cps),
		Kept:       len(cps) - len(toDelete),
		Deleted:    len(toDelete),
		DeletedIDs: toDelete,
	}

	if dryRun || len(toDelete) == 0 {
		return res, nil
	}

	if err := db.DeleteCheckpointsByID(ctx.DB, ctx.WorkspaceID, toDelete); err != nil {
		return PruneResult{}, err
	}

	return res, nil
}
