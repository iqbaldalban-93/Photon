package gphotos

import (
	"encoding/base64"
	"log/slog"

	"app/internal/protox"
)

type Item struct {
	MediaKey          string   `json:"mediaKey"`
	DedupKey          string   `json:"dedupKey"`
	FileName          string   `json:"fileName"`
	Caption           string   `json:"caption,omitempty"`
	Type              string   `json:"type"`
	TimestampUTCMS    int64    `json:"timestampMs"`
	TimezoneOffsetSec int64    `json:"tzOffset"`
	ServerCreateTS    int64    `json:"serverCreateTs"`
	SizeBytes         int64    `json:"size"`
	Width             int64    `json:"width,omitempty"`
	Height            int64    `json:"height,omitempty"`
	DurationMS        int64    `json:"durationMs,omitempty"`
	ContentVersion    int64    `json:"contentVersion"`
	TrashTimestamp    int64    `json:"trashTimestamp"`
	QuotaChargedBytes int64    `json:"quotaChargedBytes"`
	IsArchived        bool     `json:"isArchived"`
	IsFavorite        bool     `json:"isFavorite"`
	IsLocked          bool     `json:"isLocked"`
	IsEdited          bool     `json:"isEdited"`
	IsOriginalQuality bool     `json:"isOriginalQuality"`
	CollectionIDs     []string `json:"collectionIds,omitempty"`
	Latitude          float64  `json:"latitude,omitempty"`
	Longitude         float64  `json:"longitude,omitempty"`
	LocationName      string   `json:"locationName,omitempty"`

	Watched bool `json:"watched,omitempty"`
}

type Album struct {
	Key           string `json:"key"`
	Title         string `json:"title"`
	ItemCount     int64  `json:"itemCount"`
	Type          int64  `json:"type"`
	CoverMediaKey string `json:"coverMediaKey,omitempty"`
}

type SyncPage struct {
	SyncToken   string
	ResumeToken string
	Items       []*Item
	Albums      []*Album
	Deletions   []string
	Continue    bool
}

func ParseLibraryResponse(body []byte, log *slog.Logger) (*SyncPage, error) {
	root, err := protox.Parse(body)
	if err != nil {
		return nil, err
	}
	top := root.Sub(1)
	if top == nil {
		return nil, errEmptySyncResponse
	}

	page := &SyncPage{
		SyncToken:   top.Str(6),
		ResumeToken: top.Str(1),
		Continue:    top.Uint(7) == 2,
	}

	for _, d := range top.Msgs(2) {
		if item := parseMediaItem(d); item != nil {
			page.Items = append(page.Items, item)
		}
	}
	for _, d := range top.Msgs(3) {
		if album := parseCollection(d); album != nil {
			page.Albums = append(page.Albums, album)
		}
	}
	for _, d := range top.Msgs(9) {
		if key := parseDeletion(d); key != "" {
			page.Deletions = append(page.Deletions, key)
		}
	}
	return page, nil
}

var errEmptySyncResponse = errSync("library sync response missing envelope")

type errSync string

func (e errSync) Error() string { return string(e) }

func parseMediaItem(d protox.Msg) *Item {
	mediaKey := d.Str(1)
	if mediaKey == "" {
		return nil
	}
	d2 := d.Sub(2)
	if d2 == nil {
		return nil
	}
	d5 := d.Sub(5)
	d17 := d.Sub(17)

	item := &Item{MediaKey: mediaKey}

	if d21 := d2.Sub(21); d21 != nil {
		for _, num := range []int{1, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19} {
			if v := d21.Str(num); v != "" {
				item.DedupKey = v
				break
			}
		}
	}
	if item.DedupKey == "" {
		if sha := d2.Sub(13); sha != nil {
			if h := sha.Raw(1); h != nil {
				item.DedupKey = base64.RawURLEncoding.EncodeToString(h)
			}
		}
	}
	if item.DedupKey == "" {
		item.DedupKey = mediaKey
	}

	item.FileName = d2.Str(4)
	if item.FileName == "" {
		item.FileName = mediaKey + ".bin"
	}
	item.Caption = d2.Str(3)
	item.TimestampUTCMS = int64(d2.Uint(7))
	item.TimezoneOffsetSec = int64(d2.Uint(8))
	item.ServerCreateTS = int64(d2.Uint(9))
	item.SizeBytes = int64(d2.Uint(10))
	item.ContentVersion = int64(d2.Uint(26))
	if d16 := d2.Sub(16); d16 != nil {
		item.TrashTimestamp = int64(d16.Uint(3))
	}
	if d29 := d2.Sub(29); d29 != nil {
		item.IsArchived = d29.Uint(1) == 1
	}
	if d31 := d2.Sub(31); d31 != nil {
		item.IsFavorite = d31.Uint(1) == 1
	}
	if d39 := d2.Sub(39); d39 != nil {
		item.IsLocked = d39.Uint(1) == 1
	}
	if d35 := d2.Sub(35); d35 != nil {
		item.QuotaChargedBytes = int64(d35.Uint(2))
		item.IsOriginalQuality = d35.Uint(3) == 2
	}
	for _, col := range d2.Msgs(1) {
		if id := col.Str(1); id != "" {
			item.CollectionIDs = append(item.CollectionIDs, id)
		}
	}

	if d17 != nil {
		if d171 := d17.Sub(1); d171 != nil {
			item.Latitude = protox.Fixed32ToGeo(d171.Uint(1))
			item.Longitude = protox.Fixed32ToGeo(d171.Uint(2))
		}
		if d175 := d17.Sub(5); d175 != nil {
			if d1752 := d175.Sub(2); d1752 != nil {
				item.LocationName = d1752.Str(1)
			}
		}
	}

	if d5 != nil {
		item.Type = "photo"
		if t := d5.Uint(1); t != 0 {
			item.Type = mapType(t)
		}
		if photo := d5.Sub(2); photo != nil {
			item.IsEdited = photo.Sub(4) != nil
			if p1 := photo.Sub(1); p1 != nil {
				p19 := p1.Sub(9)
				if p19 != nil {
					item.Width = int64(p19.Uint(1))
					item.Height = int64(p19.Uint(2))
				}
			}
		}
		if video := d5.Sub(3); video != nil {
			item.Type = "video"
			if v2 := video.Sub(2); v2 != nil {
				_ = v2.Str(1)
			}
			if v4 := video.Sub(4); v4 != nil {
				item.DurationMS = int64(v4.Uint(1))
				item.Width = int64(v4.Uint(4))
				item.Height = int64(v4.Uint(5))
			}
		}
		if micro := d5.Sub(5); micro != nil {
			if m2 := micro.Sub(2); m2 != nil {
				if m24 := m2.Sub(4); m24 != nil {
					item.Type = "micro-video"
					item.DurationMS = int64(m24.Uint(1))
					item.Width = int64(m24.Uint(4))
					item.Height = int64(m24.Uint(5))
				}
			}
		}
	}
	return item
}

func mapType(t uint64) string {
	switch t {
	case 2:
		return "video"
	case 1:
		return "photo"
	default:
		return "photo"
	}
}

func parseCollection(d protox.Msg) *Album {
	key := d.Str(1)
	if key == "" {
		return nil
	}
	d2 := d.Sub(2)
	if d2 == nil {
		return nil
	}
	album := &Album{
		Key:       key,
		Title:     d2.Str(5),
		ItemCount: int64(d2.Uint(7)),
		Type:      int64(d2.Uint(8)),
	}
	if d217 := d2.Sub(17); d217 != nil {
		album.CoverMediaKey = d217.Str(1)
	}
	return album
}

func parseDeletion(d protox.Msg) string {
	d1 := d.Sub(1)
	if d1 == nil {
		return ""
	}
	if d1.Uint(1) == 1 {
		if d12 := d1.Sub(2); d12 != nil {
			return d12.Str(1)
		}
	}
	return ""
}
