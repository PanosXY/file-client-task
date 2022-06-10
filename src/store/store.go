package store

import (
	"io"
	"sync"
)

type fileInfo struct {
	charIndex int
	content   *io.ReadCloser
}

type FileStorage struct {
	mutex sync.RWMutex
	files map[string]*fileInfo
}

func NewFileStorage() *FileStorage {
	fs := new(FileStorage)
	fs.files = make(map[string]*fileInfo)
	return fs
}

func (fs *FileStorage) NewFile(filename string) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	fs.files[filename] = new(fileInfo)
	fs.files[filename].charIndex = -1
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
	fs.files[filename].content = content
}

func (fs *FileStorage) SetFileCharIndex(filename string, idx int) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	fs.files[filename].charIndex = idx
}

func (fs *FileStorage) DeleteFile(filename string) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	delete(fs.files, filename)
}

func (fs *FileStorage) GetFileContent(filename string) *io.ReadCloser {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	return fs.files[filename].content
}

func (fs *FileStorage) GetFileCharIndex(filename string) int {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	return fs.files[filename].charIndex
}
