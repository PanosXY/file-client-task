package store

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
)

type fileShards []string

type FileStorage struct {
	mutex     sync.RWMutex
	files     map[string]fileShards
	chunkSize uint64
}

func NewFileStorage(cs uint64) *FileStorage {
	fs := new(FileStorage)
	fs.files = make(map[string]fileShards)
	fs.chunkSize = cs
	return fs
}

func (fs *FileStorage) NewFile(filename string) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	fs.files[filename] = make(fileShards, 0)
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

func (fs *FileStorage) StoreFileShard(dir, filename string, shard uint64, buf []byte) error {
	targetFile := fmt.Sprintf("%s/%s%d.tmp", dir, filename, shard)
	file, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("Failed to create tmp file for '%s': %v", filename, err)
	}
	defer file.Close()
	if _, err := io.Copy(file, bytes.NewReader(buf)); err != nil {
		return fmt.Errorf("Failed to save tmp file for '%s': %v", filename, err)
	}
	fs.setShard(filename, targetFile)
	return nil
}

func (fs *FileStorage) setShard(filename, shard string) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	fs.files[filename] = append(fs.files[filename], shard)
}

func (fs *FileStorage) SaveFiles(path, zipFilename string, filenames []string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	defer fs.deleteFileShards()
	if len(filenames) == 0 {
		return fmt.Errorf("There are no files to save")
	}
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
	file, err := fs.joinShards(filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, bytes.NewReader(file)); err != nil {
		return err
	}
	return nil
}

func (fs *FileStorage) joinShards(filename string) ([]byte, error) {
	file := make([]byte, 0, uint64(len(fs.files[filename]))*fs.chunkSize)
	for _, shard := range fs.files[filename] {
		buf, err := os.ReadFile(shard)
		if err != nil {
			return nil, err
		}
		file = append(file, buf...)
	}
	return file, nil
}

func (fs *FileStorage) deleteFileShards() {
	for _, shards := range fs.files {
		for _, v := range shards {
			os.Remove(v)
		}
	}
}
