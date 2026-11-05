package gphotos

import (
	"errors"
	"fmt"

	"app/internal/credentials"
	gphotoclient "app/internal/gphotos"
	"app/libweb/configx"
)

// Api is the browser-library handle onto the shared Google Photos client.
// Upload/commit/album endpoints are promoted from the embedded client; the
// library-sync, mutation and download endpoints live in sibling files and
// ride the shared transport via the promoted DoProto.

// Endpoints used only by the library-web surface.
const (
	defaultMaxConnsPerHost = 10

	deleteAlbumEndpoint     = "https://photosdata-pa.googleapis.com/6439526531001121323/11165707358190966680"
	renameAlbumEndpoint     = "https://photosdata-pa.googleapis.com/6439526531001121323/16466587394238175348"
	trashEndpoint           = "https://photosdata-pa.googleapis.com/6439526531001121323/17490284929287180316"
	archiveEndpoint         = "https://photosdata-pa.googleapis.com/6439526531001121323/6715446385130606868"
	favoriteEndpoint        = "https://photosdata-pa.googleapis.com/6439526531001121323/5144645502632292153"
	captionEndpoint         = "https://photosdata-pa.googleapis.com/6439526531001121323/1552790390512470739"
	librarySyncEndpoint     = "https://photosdata-pa.googleapis.com/6439526531001121323/18047484249733410717"
	prepareDownloadEndpoint = "https://photosdata-pa.googleapis.com/$rpc/social.frontend.photos.preparedownloaddata.v1.PhotosPrepareDownloadDataService/PhotosPrepareDownload"
)

type Api struct {
	*gphotoclient.Client
}

func NewApi() (*Api, error) {
	cfg := configx.Get()
	selected := cfg.Photos.Selected
	if selected == "" {
		return nil, errors.New("no account is selected")
	}
	cred := credentials.Select(cfg.Photos.Credentials, selected)
	if cred == "" {
		return nil, fmt.Errorf("no credentials with matching selected email found")
	}
	return NewApiWithCredentials(cred)
}

func NewApiWithCredentials(cred string) (*Api, error) {
	cfg := configx.Get()
	client, err := gphotoclient.NewClient(gphotoclient.Options{
		Credentials:     cred,
		Proxy:           cfg.Photos.Proxy,
		MaxConnsPerHost: defaultMaxConnsPerHost,
	})
	if err != nil {
		return nil, err
	}
	return &Api{Client: client}, nil
}
