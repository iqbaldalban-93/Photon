package server

import (
	"net/http"
	"strings"

	"app/internal/proxyurl"
	"app/libweb/configx"
)

// Library-web settings endpoints.

func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	jsonWrite(w, http.StatusOK, redactPhotos(configx.Get().Photos))
}

func (s *Server) handleSettingsPut(w http.ResponseWriter, r *http.Request) {
	var body configx.Photos
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	if body.UploadThreads < 1 {
		body.UploadThreads = 1
	}
	body.UploadThreads = min(body.UploadThreads, configx.MaxUploadThreads)
	current := configx.Get().Photos
	body.Credentials = current.Credentials
	body.Selected = current.Selected
	if !isLoopbackRequest(r) {
		body.DeleteFromHost = current.DeleteFromHost
		body.DisableUnsupportedFilesFilter = current.DisableUnsupportedFilesFilter
		body.Proxy = current.Proxy
	}
	if _, err := proxyurl.Parse(body.Proxy); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	body.Proxy = strings.TrimSpace(body.Proxy)
	if err := configx.Update(func(c *configx.Config) { c.Photos = body }); err != nil {
		writeErr(w, err)
		return
	}
	jsonWrite(w, http.StatusOK, redactPhotos(configx.Get().Photos))
}
