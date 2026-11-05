package uploadcore

import "fmt"

// Album batch planning shared by the desktop AlbumManager and the web
// pinToAlbum: chunk media keys into add-batches and split sets larger than
// the album item cap into numbered albums ("Name (2)", "Name (3)", …).

const (
	// AlbumBatchSize is the number of media keys sent per AddMediaToAlbum call.
	AlbumBatchSize = 500
	// AlbumLimit is Google Photos' per-album item cap.
	AlbumLimit = 20000
)

// AlbumChunk is one API call's worth of work: Batch (<= AlbumBatchSize keys)
// targeting album Name. IsFirstOfAlbum marks the chunk that must create the
// album instead of adding to it.
type AlbumChunk struct {
	Name           string
	Batch          []string
	IsFirstOfAlbum bool
}

// PlanAlbumBatches lays out mediaKeys into album-creation and add batches.
func PlanAlbumBatches(mediaKeys []string, baseName string) []AlbumChunk {
	var plan []AlbumChunk
	albumCounter := 0
	split := len(mediaKeys) > AlbumLimit

	for i := 0; i < len(mediaKeys); i += AlbumLimit {
		end := min(i+AlbumLimit, len(mediaKeys))
		albumCounter++
		name := baseName
		if split {
			name = fmt.Sprintf("%s (%d)", baseName, albumCounter)
		}
		for j := i; j < end; j += AlbumBatchSize {
			batchEnd := min(j+AlbumBatchSize, end)
			plan = append(plan, AlbumChunk{
				Name:           name,
				Batch:          mediaKeys[j:batchEnd],
				IsFirstOfAlbum: j == i,
			})
		}
	}
	return plan
}
