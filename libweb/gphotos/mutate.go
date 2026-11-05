package gphotos

import (
	"context"

	"app/internal/protox"
)

const (
	clientVersionCodeInt = 49029607
	androidAPIVersionStr = "28"
)

func (a *Api) TrashAction(ctx context.Context, dedupKeys []string) error {
	return a.trashRPC(ctx, trashOp{
		Op:     1,
		Flag:   1,
		Keys:   dedupKeys,
		Mask:   map[string]any{"4": map[string]any{"2": map[string]any{}, "3": map[string]any{"1": map[string]any{}}, "4": map[string]any{}, "5": map[string]any{"1": map[string]any{}}}},
		Caller: map[string]any{"1": 5, "2": map[string]any{"1": clientVersionCodeInt, "2": androidAPIVersionStr}},
	})
}

func (a *Api) RestoreFromTrash(ctx context.Context, dedupKeys []string) error {
	return a.trashRPC(ctx, trashOp{
		Op:     3,
		Flag:   2,
		Keys:   dedupKeys,
		Mask:   map[string]any{"4": map[string]any{"2": map[string]any{}, "3": map[string]any{"1": map[string]any{}}}},
		Caller: map[string]any{"1": 5, "2": map[string]any{"1": clientVersionCodeInt, "2": androidAPIVersionStr}},
	})
}

func (a *Api) DeletePermanently(ctx context.Context, dedupKeys []string) error {
	return a.trashRPC(ctx, trashOp{
		Op:     2,
		Flag:   2,
		Keys:   dedupKeys,
		Mask:   map[string]any{"4": map[string]any{"2": map[string]any{}, "3": map[string]any{"1": map[string]any{}}, "4": map[string]any{}, "5": map[string]any{"1": map[string]any{}}}},
		Caller: map[string]any{},
	})
}

type trashOp struct {
	Op     int
	Flag   int
	Keys   []string
	Mask   map[string]any
	Caller map[string]any
}

func (a *Api) trashRPC(ctx context.Context, op trashOp) error {
	body := map[string]any{
		"2": op.Op,
		"3": op.Keys,
		"4": op.Flag,
		"8": op.Mask,
		"9": op.Caller,
	}
	_, err := a.DoProto(ctx, trashEndpoint, protox.FromMap(body), false)
	return err
}

func (a *Api) SetArchived(ctx context.Context, dedupKeys []string, archived bool) error {
	action := 2
	if archived {
		action = 1
	}
	var items []any
	for _, k := range dedupKeys {
		items = append(items, map[string]any{"1": k, "2": map[string]any{"1": action}})
	}
	body := map[string]any{"1": items, "3": 1}
	_, err := a.DoProto(ctx, archiveEndpoint, protox.FromMap(body), true)
	return err
}

func (a *Api) SetFavorite(ctx context.Context, dedupKey string, favorite bool) error {
	action := 2
	if favorite {
		action = 1
	}
	body := map[string]any{
		"1": map[string]any{"2": dedupKey},
		"2": map[string]any{"1": action},
		"3": map[string]any{"1": map[string]any{"19": map[string]any{}}},
	}
	_, err := a.DoProto(ctx, favoriteEndpoint, protox.FromMap(body), true)
	return err
}

func (a *Api) SetCaption(ctx context.Context, dedupKey, caption string) error {
	body := map[string]any{"2": caption, "3": dedupKey}
	_, err := a.DoProto(ctx, captionEndpoint, protox.FromMap(body), true)
	return err
}
