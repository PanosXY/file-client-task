package store

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
)

type FileStorage struct {
	mutex sync.RWMutex
	files map[string][]byte
}

func NewFileStorage() *FileStorage {
	fs := new(FileStorage)
	fs.files = make(map[string][]byte)
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

func (fs *FileStorage) SetFileContent(filename string, content []byte) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	fs.files[filename] = content
}

func (fs *FileStorage) SaveFiles(path, zipFilename string, filenames []string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	file, err := os.Create(path + "/" + zipFilename)
	if err != nil {
		return fmt.Errorf("Failed to create zip file: %v", err)
	}
	defer file.Close()

	zipw := zip.NewWriter(file)
	defer zipw.Close()

	for _, filename := range filenames {
		if err := fs.appendFiles(filename, zipw); err != nil {
			return fmt.Errorf("Failed to add file %s to zip: %s", filename, err)
		}
	}
	return nil
}

func (fs *FileStorage) appendFiles(filename string, zipw *zip.Writer) error {
	w, err := zipw.Create(filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, bytes.NewReader(fs.files[filename])); err != nil {
		return err
	}
	return nil
}
