package client

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/panosxy/file-client-task/downloader"
	"github.com/panosxy/file-client-task/store"
	"github.com/panosxy/file-client-task/utils"
)

type client struct {
	wg         sync.WaitGroup
	files      *store.FileStorage
	downloader *downloader.ConcurrentDownloader
	url        string
	char       []byte
	dlResult   chan *downloader.FileContent
	dlDone     chan struct{}
	log        *utils.Logger
}

func NewClient(url, char string, workers uint, log *utils.Logger) (*client, error) {
	c := new(client)
	c.wg = sync.WaitGroup{}
	c.files = store.NewFileStorage()
	c.downloader = downloader.NewConcurrentDownloader(log, workers)
	c.url = url
	if len([]rune(char)) != 1 {
		return nil, fmt.Errorf("'%s' is not a signle character", char)
	}
	c.char = []byte(char)
	c.dlResult = make(chan *downloader.FileContent, workers)
	c.dlDone = make(chan struct{})
	c.log = log
	return c, nil
}

func (c *client) Do() error {
	// List files
	if err := c.getFiles(); err != nil {
		return fmt.Errorf("Error on getting the files from the server: %v", err)
	}
	// Get files
	if err := c.downloader.Subscribe(c.url, c.files.GetFilesnames(), c.dlResult, c.dlDone); err != nil {
		return fmt.Errorf("Error on downloader subscription: %v", err)
	}
	c.downloader.Start()

DownloadLoop:
	for {
		select {
		case fc := <-c.dlResult:
			c.files.SetFileContent(fc.Filename, fc.Content)
		case <-c.dlDone:
			c.log.Info("Download done!")
			break DownloadLoop
		}
	}
	return nil
}

func (c *client) getFiles() error {
	files, err := listPath(c.url)
	if err != nil {
		return fmt.Errorf("Could't list '%s' url: %v", c.url, err)
	}
	if len(files) == 0 {
		return fmt.Errorf("Requested path is empty")
	}
	c.log.Info(fmt.Sprintf("Retrieved files: %v\n", files))
	for _, filename := range files {
		c.files.NewFile(filename)
	}
	return nil
}

func listPath(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, err
	}
	return utils.GetLinks(resp.Body), nil
}
