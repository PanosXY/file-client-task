package downloader

import (
	"io"
	"testing"
	"time"

	"github.com/PanosXY/file-client-task/utils"
	"github.com/stretchr/testify/assert"
)

func TestDownloaderSubscribe(t *testing.T) {
	d := NewConcurrentDownloader(nil, 4)

	url := "http://google.com"
	files := []string{"t"}
	handler := func(s string, rc io.ReadCloser) error {
		return nil
	}
	// Test Subscribe with various passed parameters
	assert.NoError(t, d.Subscribe(url, files, handler))
	assert.Error(t, d.Subscribe("", files, handler))
	assert.Error(t, d.Subscribe(url, []string{}, handler))
	assert.Error(t, d.Subscribe(url, files, nil))
}

func TestDownloaderDownload(t *testing.T) {
	log, _ := utils.NewLogger(true)
	d := NewConcurrentDownloader(log, 4)

	// Test with no subscription
	assert.Error(t, d.Start())
	// Test with already started
	url := "http://google.com"
	files := []string{"t"}
	handler := func(s string, rc io.ReadCloser) error {
		time.Sleep(1000)
		return nil
	}
	assert.NoError(t, d.Subscribe(url, files, handler))
	assert.NoError(t, d.Start())
	err := d.Start()
	assert.Error(t, err, err)
	// Test with re-start with no subscription
	time.Sleep(1500)
	err = d.Start()
	assert.Error(t, err, err)
}
