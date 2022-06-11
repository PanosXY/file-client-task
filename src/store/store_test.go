package store

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStorageSequential(t *testing.T) {
	s := NewFileStorage()
	// Cenerate 4 random filenames and their content
	filenames := []string{"t1.txt", "t2.txt", "t3.txt", "t4.txt"}
	content := [][]byte{[]byte("Hello world"), []byte("file-client"), []byte("task"), []byte("Acronis")}

	for i, filename := range filenames {
		s.NewFile(filename)
		s.SetFileContent(filename, content[i])
	}
	// Check that the storage contains 4 files
	assert.Equal(t, s.Len(), 4)
	// Check for data validity
	assert.ElementsMatch(t, s.GetFilesnames(), filenames)
	for i, filename := range filenames {
		assert.ElementsMatch(t, s.GetFileContent(filename), content[i])
	}
	// Check SaveFile()
	assert.NoError(t, s.SaveFiles("/tmp", "test.zip", filenames))
	assert.Error(t, s.SaveFiles("/tmp", "test.zip", []string{})) // Pass empty slice
	assert.Error(t, s.SaveFiles("/usr", "test.zip", filenames))  // Path without permissions
}

func TestFileStorageConcurrency(t *testing.T) {
	s := NewFileStorage()

	// Spawn 5 goroutines. Each goroutine will add 10000 files.
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 10000; j++ {
				filename := fmt.Sprintf("test%d-%d", i, j)
				s.NewFile(filename)
				s.SetFileContent(filename, []byte("test"))
			}
		}(i)
	}
	wg.Wait()
	require.Equal(t, s.Len(), 5*10000)
}

func BenchmarkWriteStorage(b *testing.B) {
	s := NewFileStorage()

	for i := 0; i < b.N; i++ {
		filename := fmt.Sprintf("test%d", i)
		s.NewFile(filename)
		s.SetFileContent(filename, []byte("Lorem ipsum"))
	}
}
