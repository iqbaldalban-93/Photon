package gphotos

import (
	"sync"

	"app/internal/credentials"
	"app/libweb/configx"
)

var (
	sharedMu  sync.Mutex
	sharedApi *Api
	sharedKey string
	sharedOK  bool
)

func Shared() (*Api, error) {
	cfg := configx.Get()
	selected := cfg.Photos.Selected
	cred := ""
	if selected != "" {
		cred = credentials.Select(cfg.Photos.Credentials, selected)
	}

	key := selected + "\x00" + cred
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if sharedOK && sharedKey == key {
		return sharedApi, nil
	}
	api, err := NewApi()
	if err != nil {
		if sharedOK && selected == "" {
			return sharedApi, nil
		}
		return nil, err
	}
	sharedApi, sharedKey, sharedOK = api, key, true
	return api, nil
}
