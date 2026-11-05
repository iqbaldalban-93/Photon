package gphotos

import (
	"context"
	"errors"
	"fmt"
	"time"

	"app/internal/protox"
)

// Album mutation endpoints.

func (c *Client) CreateAlbum(ctx context.Context, albumName string, mediaKeys []string) (string, error) {
	b := protox.NewBuilder()
	b.Str(1, albumName)
	b.Int(2, time.Now().Unix())
	b.Int(3, 1)
	b.Each(4, toAnySlice(mediaKeys), func(m *protox.Builder, key any) {
		m.Msg(1, func(k *protox.Builder) { k.Str(1, key.(string)) })
	})
	b.Msg(6, func(*protox.Builder) {})
	b.Msg(7, func(f7 *protox.Builder) { f7.Int(1, 3) })
	b.Msg(8, func(f8 *protox.Builder) {
		f8.Str(3, defaultModel)
		f8.Str(4, defaultMake)
		f8.Int(5, androidAPIVersionInt64)
	})

	respBody, err := c.DoProto(ctx, CreateAlbumEndpoint, b.Bytes_(), true)
	if err != nil {
		return "", err
	}

	msg, err := protox.Parse(respBody)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal protobuf: %w", err)
	}
	f1 := msg.Sub(1)
	if f1 == nil {
		return "", errors.New("create album failed: invalid response structure")
	}
	albumMediaKey := f1.Str(1)
	if albumMediaKey == "" {
		return "", errors.New("create album failed: no album media key returned")
	}
	return albumMediaKey, nil
}

func (c *Client) AddMediaToAlbum(ctx context.Context, albumMediaKey string, mediaKeys []string) error {
	b := protox.NewBuilder()
	for _, key := range mediaKeys {
		b.Str(1, key)
	}
	b.Str(2, albumMediaKey)
	b.Msg(5, func(f5 *protox.Builder) { f5.Int(1, albumActionAdd) })
	b.Msg(6, func(f6 *protox.Builder) {
		f6.Str(3, defaultModel)
		f6.Str(4, defaultMake)
		f6.Int(5, androidAPIVersionInt64)
	})
	b.Int(7, time.Now().Unix())

	_, err := c.DoProto(ctx, AddMediaToAlbumEndpoint, b.Bytes_(), true)
	return err
}

func (c *Client) RemoveMediaFromAlbum(ctx context.Context, albumMediaKey string, mediaKeys []string) error {
	b := protox.NewBuilder()
	for _, key := range mediaKeys {
		b.Str(1, key)
	}
	b.Str(2, albumMediaKey)
	b.Msg(5, func(f5 *protox.Builder) { f5.Int(1, albumActionRemove) })
	b.Msg(6, func(f6 *protox.Builder) {
		f6.Str(3, defaultModel)
		f6.Str(4, defaultMake)
		f6.Int(5, androidAPIVersionInt64)
	})
	b.Int(7, time.Now().Unix())

	_, err := c.DoProto(ctx, AddMediaToAlbumEndpoint, b.Bytes_(), true)
	return err
}

func (c *Client) DeleteAlbum(ctx context.Context, albumMediaKey string) error {
	b := protox.NewBuilder().Str(1, albumMediaKey)
	_, err := c.DoProto(ctx, DeleteAlbumEndpoint, b.Bytes_(), true)
	return err
}

func (c *Client) RenameAlbum(ctx context.Context, albumMediaKey, newName string) error {
	b := protox.NewBuilder().Str(1, albumMediaKey).Str(2, newName).Int(3, 1)
	_, err := c.DoProto(ctx, RenameAlbumEndpoint, b.Bytes_(), true)
	return err
}

const (
	albumActionRemove = 1
	albumActionAdd    = 2
)

func toAnySlice(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}
