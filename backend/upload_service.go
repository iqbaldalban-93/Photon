package backend

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type UploadService struct{}

func (s *UploadService) ListAlbums() ([]AlbumInfo, error) {
	api, err := NewApi()
	if err != nil {
		return nil, err
	}
	return api.ListAlbums()
}

func (s *UploadService) SelectFiles() ([]string, error) {
	app := application.Get()
	if app == nil {
		return nil, fmt.Errorf("application is not running")
	}
	paths, err := app.Dialog.OpenFile().
		CanChooseFiles(true).
		CanChooseDirectories(false).
		SetTitle("Select files to upload").
		PromptForMultipleSelection()
	if err != nil {
		return nil, fmt.Errorf("failed to open file dialog: %w", err)
	}
	return paths, nil
}

func (s *UploadService) SelectFolder() (string, error) {
	app := application.Get()
	if app == nil {
		return "", fmt.Errorf("application is not running")
	}
	path, err := app.Dialog.OpenFile().
		CanChooseFiles(false).
		CanChooseDirectories(true).
		SetTitle("Select a folder to upload").
		PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("failed to open folder dialog: %w", err)
	}
	return path, nil
}
