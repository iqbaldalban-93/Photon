package backend

import "app/internal/hostdelete"

func deleteFileFromHost(path string) error { return hostdelete.Trash(path) }
