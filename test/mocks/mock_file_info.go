package mocks

import (
	"io/fs"
	"time"
)

type MockFileInfo struct {
}

func (m MockFileInfo) Name() string {
	return "test name"
}

func (m MockFileInfo) Size() int64 {
	return 42
}

func (m MockFileInfo) Mode() fs.FileMode {
	//TODO implement me
	panic("implement me")
}

func (m MockFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (m MockFileInfo) IsDir() bool {
	return false
}

func (m MockFileInfo) Sys() interface{} {
	return true
}
