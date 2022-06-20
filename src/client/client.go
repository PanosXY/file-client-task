package client

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/PanosXY/file-client-task/downloader"
	"github.com/PanosXY/file-client-task/store"
	"github.com/PanosXY/file-client-task/utils"
)

const (
	zipFilename string = "file-client-task.zip"
	tmpDir      string = "/tmp"
	chunkSize   int64  = 4 * 1024
)

type indexHandler struct {
	sync.Mutex
	index int64
}

func newIndexHandler() *indexHandler {
	ih := new(indexHandler)
	ih.index = -1
	return ih
}

type client struct {
	mutex        sync.Mutex
	wg           sync.WaitGroup
	files        *store.FileStorage
	downloader   *downloader.ConcurrentDownloader
	url          string
	char         rune
	minCharIndex *indexHandler
	filesToDl    map[int64][]string
	dlPath       string
	log          *utils.Logger
}

func NewClient(url, char string, workers uint, dlPath string, log *utils.Logger) (*client, error) {
	c := new(client)
	c.files = store.NewFileStorage(int(chunkSize))
	c.downloader = downloader.NewConcurrentDownloader(log, workers)
	c.url = url
	if len([]rune(char)) != 1 {
		return nil, fmt.Errorf("'%s' is not a signle character", char)
	}
	c.char = []rune(char)[0]
	c.minCharIndex = newIndexHandler()
	c.filesToDl = make(map[int64][]string)
	c.dlPath = dlPath
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
	// Download file(s)
	if c.minCharIndex.index == -1 {
		c.log.Info(fmt.Sprintf("No files found including character '%s' on url '%s'", string(c.char), c.url))
		return nil
	}
	if err := c.files.SaveFiles(c.dlPath, zipFilename, c.filesToDl[c.minCharIndex.index]); err != nil {
		return fmt.Errorf("Error on downloading file(s): %v", err)
	}
	c.log.Info(fmt.Sprintf("File(s) downloaded successfully in '%s'", c.dlPath+"/"+zipFilename))
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
	if err := c.downloader.Subscribe(c.url, c.files.GetFilesnames(), c.scanAndStore); err != nil {
		return fmt.Errorf("Error on downloader subscription: %v", err)
	}
	c.log.Info("Downloading...")
	c.downloader.Start()
	c.log.Info("Fetching files' content done!")
	return nil
}

func (c *client) scanAndStore(filename string, content io.ReadCloser) error {
	chunks := int64(0)
	reader := bufio.NewReader(content)
	buf := make([]byte, 0, chunkSize)
	charFound := false
	for {
		n, err := reader.Read(buf[:cap(buf)])
		buf = buf[:n]
		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}
			return err
		}
		chunks++
		if err != nil && err != io.EOF {
			return err
		}
		// Get index
		if omit := c.getIndex(filename, buf, chunks, &charFound); omit {
			break
		}
		// Store
		if err := c.files.StoreFileShard(tmpDir, filename, int(chunks), buf); err != nil {
			return err
		}
	}
	return nil
}

func (c *client) getIndex(filename string, buf []byte, chunk int64, charFound *bool) bool {
	if *charFound {
		return false
	}
	// XXX: If a character is more than one byte and it happens to
	//      be split by the chunks (i.e. character 'Я' at index 4095),
	//      the below code does not work
	if idx := bytes.IndexRune(buf, c.char); idx != -1 {
		aggregatedIdx := int64(idx) + ((chunk - 1) * chunkSize)
		c.minCharIndex.Lock()
		if c.minCharIndex.index != -1 && aggregatedIdx > c.minCharIndex.index {
			c.minCharIndex.Unlock()
			return true
		}
		c.minCharIndex.index = aggregatedIdx
		c.minCharIndex.Unlock()
		c.fileToDownload(aggregatedIdx, filename)
		*charFound = true
	}
	return false
}

func (c *client) fileToDownload(idx int64, filename string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if _, ok := c.filesToDl[idx]; !ok {
		c.filesToDl[idx] = []string{filename}
	} else {
		c.filesToDl[idx] = append(c.filesToDl[idx], filename)
	}
}
