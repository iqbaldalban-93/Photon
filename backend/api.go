package backend

import (
	"context"
	"fmt"
	"net/http"

	"app/internal/credentials"
	gphotoclient "app/internal/gphotos"
)

// Api is the desktop layer's handle onto the shared Google Photos client.
// Credential selection and storage-quality mapping stay here; the wire
// protocol lives in internal/gphotos.

type Api struct {
	account string
	client  *gphotoclient.Client
}

func NewApi() (*Api, error) {
	return NewApiForAccount(CurrentConfig().Selected)
}

func NewApiForAccount(selectedEmail string) (*Api, error) {
	cfg := CurrentConfig()
	if len(selectedEmail) == 0 {
		return nil, fmt.Errorf("no account is selected")
	}
	cred := credentials.Select(cfg.Credentials, selectedEmail)
	if len(cred) == 0 {
		return nil, fmt.Errorf("no credentials with matching selected email found")
	}

	client, err := gphotoclient.NewClient(gphotoclient.Options{
		Credentials:     cred,
		Proxy:           cfg.Proxy,
		MaxConnsPerHost: maxConnsPerHost,
	})
	if err != nil {
		return nil, err
	}
	return &Api{account: selectedEmail, client: client}, nil
}

func (a *Api) BearerToken() (string, error) {
	return a.client.BearerToken()
}

func newAPIFromCredential(cred string, proxy string) (*Api, error) {
	client, err := gphotoclient.NewClient(gphotoclient.Options{
		Credentials:     cred,
		Proxy:           proxy,
		MaxConnsPerHost: maxConnsPerHost,
	})
	if err != nil {
		return nil, err
	}
	return &Api{account: credentials.Email(cred), client: client}, nil
}

// HTTPClient exposes the underlying *http.Client (e.g. to install redirect
// policy for the credential-validation flow).
func (a *Api) HTTPClient() *http.Client {
	return a.client.HTTP()
}

func (a *Api) GetUploadToken(ctx context.Context, shaHashB64 string, fileSize int64) (string, error) {
	return a.client.GetUploadToken(ctx, shaHashB64, fileSize)
}

func (a *Api) FindRemoteMediaByHash(ctx context.Context, shaHash []byte) (string, error) {
	return a.client.FindRemoteMediaByHash(ctx, shaHash)
}

func (a *Api) UploadFileWithProgress(ctx context.Context, filePath string, uploadToken string, onProgress gphotoclient.UploadProgressCallback) (gphotoclient.ScottyFinalizeToken, error) {
	return a.client.UploadFileWithProgress(ctx, filePath, uploadToken, onProgress)
}

// CommitUploadAs commits under an explicit device identity (delegated to the
// shared client's retried commit).
func (a *Api) CommitUploadAs(
	ctx context.Context,
	finalizeToken gphotoclient.ScottyFinalizeToken,
	fileName string,
	sha1Hash []byte,
	uploadTimestamp int64,
	device gphotoclient.CommitDevice,
) (string, error) {
	return a.client.CommitUploadAs(ctx, finalizeToken, fileName, sha1Hash, uploadTimestamp, device)
}

// listAlbumsRaw fetches the raw library state using the desktop layer's
// hand-rolled request (kept byte-stable by a golden test).
func (a *Api) listAlbumsRaw() ([]byte, error) {
	return a.client.DoProto(context.Background(), libraryStateEndpoint, buildLibraryStateRequest(), true)
}

func (a *Api) ListAlbums() ([]AlbumInfo, error) {
	bodyBytes, err := a.listAlbumsRaw()
	if err != nil {
		return nil, err
	}
	albums, err := parseAlbumsFromLibraryState(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse library state: %w", err)
	}
	return albums, nil
}

func (a *Api) CreateAlbum(ctx context.Context, albumName string, mediaKeys []string) (string, error) {
	return a.client.CreateAlbum(ctx, albumName, mediaKeys)
}

func (a *Api) AddMediaToAlbum(ctx context.Context, albumMediaKey string, mediaKeys []string) error {
	return a.client.AddMediaToAlbum(ctx, albumMediaKey, mediaKeys)
}
