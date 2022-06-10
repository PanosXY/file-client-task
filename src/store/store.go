package store

import (
	"io"
	"sync"
)

type FileStorage struct {
	mutex sync.RWMutex
	files map[string]*io.ReadCloser
}

func NewFileStorage() *FileStorage {
	fs := new(FileStorage)
	fs.files = make(map[string]*io.ReadCloser)
	return fs
}

func (fs *FileStorage) NewFile(filename string) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	fs.files[filename] = nil
}

func (fs *FileStorage) Len() int {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	return len(fs.files)
}

func (fs *FileStorage) GetFilesnames() []string {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	filenames := make([]string, 0)
	for k := range fs.files {
		filenames = append(filenames, k)
	}
	return filenames
}

func (fs *FileStorage) SetFileContent(filename string, content *io.ReadCloser) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	fs.files[filename] = content
}

func (fs *FileStorage) DeleteFile(filename string) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	delete(fs.files, filename)
}

func (fs *FileStorage) GetFileContent(filename string) *io.ReadCloser {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	return fs.files[filename]
}
