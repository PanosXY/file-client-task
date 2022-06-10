package client

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/PanosXY/file-client-task/downloader"
	"github.com/PanosXY/file-client-task/store"
	"github.com/PanosXY/file-client-task/utils"
)

type indexHandler struct {
	sync.Mutex
	index int
}

func newIndexHandler() *indexHandler {
	ih := new(indexHandler)
	ih.index = -1
	return ih
}

type client struct {
	wg           sync.WaitGroup
	files        *store.FileStorage
	downloader   *downloader.ConcurrentDownloader
	url          string
	char         []byte
	dlDone       chan struct{}
	minCharIndex *indexHandler
	log          *utils.Logger
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
	c.dlDone = make(chan struct{})
	c.minCharIndex = newIndexHandler()
	c.log = log
	return c, nil
}

func (c *client) Do() error {
	// List files
	if err := c.listFiles(); err != nil {
		return fmt.Errorf("Error on listing the files from the server: %v", err)
	}
	// Get files & char index
	if err := c.getFiles(); err != nil {
		return fmt.Errorf("Error on getting the files from the server: %v", err)
	}
	// TODO: Download file(s)
	return nil
}

func (c *client) listFiles() error {
	files, err := listPath(c.url)
	if err != nil {
		return fmt.Errorf("Could't list '%s' url: %v", c.url, err)
	}
	if len(files) == 0 {
		return fmt.Errorf("Requested path is empty")
	}
	c.log.Info(fmt.Sprintf("Files list: %v", files))
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

func (c *client) getFiles() error {
	if err := c.downloader.Subscribe(c.url, c.files.GetFilesnames(), c.dlDone, c.storeAndScan); err != nil {
		return fmt.Errorf("Error on downloader subscription: %v", err)
	}
	c.downloader.Start()
DownloadLoop:
	for {
		select {
		case <-c.dlDone:
			c.log.Info("Fetching files' content done!")
			break DownloadLoop
		}
	}
	return nil
}

func (c *client) storeAndScan(filename string, content io.ReadCloser) error {
	// Store
	cnt := ioutil.NopCloser(content)
	c.files.SetFileContent(filename, &cnt)
	// Scan for index
	scanner := bufio.NewScanner(content)
	scanner.Split(bufio.ScanRunes)
	c.minCharIndex.Lock()
	for i := 0; scanner.Scan(); i++ {
		if c.minCharIndex.index != -1 && i > c.minCharIndex.index {
			break
		}
		if bytes.Compare(scanner.Bytes(), c.char) == 0 {
			c.minCharIndex.index = i
			break
		}
	}
	c.minCharIndex.Unlock()
	return nil
}
