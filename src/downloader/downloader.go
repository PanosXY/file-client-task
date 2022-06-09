package downloader

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/panosxy/file-client-task/types"
	"github.com/panosxy/file-client-task/utils"
)

// A non blocking concurrent downloader

type downloaderState int

const (
	downloaderIdle downloaderState = iota
	downloaderStarted
)

type ConcurrentDownloader struct {
	wg       sync.WaitGroup
	mutex    sync.Mutex
	url      string
	queue    []string
	resultCh chan *types.FileContent
	doneCh   chan struct{}
	state    downloaderState
	workers  uint
	log      *utils.Logger
}

func NewConcurrentDownloader(log *utils.Logger, workers uint) *ConcurrentDownloader {
	d := new(ConcurrentDownloader)
	d.wg = sync.WaitGroup{}
	d.mutex = sync.Mutex{}
	d.log = log
	d.workers = workers
	d.state = downloaderIdle
	return d
}

func (d *ConcurrentDownloader) Subscribe(url string, queue []string, resultCh chan *types.FileContent, doneCh chan struct{}) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if url == "" {
		return errors.New("The subscribed url is empty")
	}
	d.url = url
	if len(queue) == 0 {
		return errors.New("The subscribed filenames queue is empty")
	}
	d.queue = queue
	if resultCh == nil {
		return errors.New("The subscribed result channel is nil")
	}
	d.resultCh = resultCh
	if doneCh == nil {
		return errors.New("The subscribed done channel is nil")
	}
	d.doneCh = doneCh
	return nil
}

func (d *ConcurrentDownloader) Start() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.state == downloaderStarted {
		return errors.New("Is already started")
	}
	if len(d.queue) == 0 {
		return errors.New("Not subscribed yet")
	}
	d.state = downloaderStarted
	go func() {
		idx := 0
		done := false
		for !done {
			for w := 0; w < min(len(d.queue), int(d.workers)); w++ {
				if idx == len(d.queue) {
					done = true
					break
				}
				d.wg.Add(1)
				go d.getFile(d.queue[idx])
				idx++
			}
			d.wg.Wait()
		}
		close(d.doneCh)
		d.cleanup()
	}()
	return nil
}

func (d *ConcurrentDownloader) getFile(filename string) {
	defer d.wg.Done()
	d.mutex.Lock()
	defer d.mutex.Unlock()
	req, err := http.NewRequest(http.MethodGet, d.url+"/"+filename, nil)
	if err != nil {
		d.log.Error(fmt.Sprintf("Couldn't download file '%s': %v", filename, err))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		d.log.Error(fmt.Sprintf("Couldn't download file '%s': %v", filename, err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		d.log.Error(fmt.Sprintf("Couldn't download file '%s': %v", filename, err))
		return
	}
	res := &types.FileContent{
		Filename: filename,
		Content:  &resp.Body,
	}
	select {
	case d.resultCh <- res:
		return
	default:
		d.log.Error("Failed to write result on subscribed channel")
		break
	}
}

func (d *ConcurrentDownloader) cleanup() {
	d.mutex.Lock()
	d.mutex.Unlock()
	d.url = ""
	d.queue = []string{}
	d.resultCh = nil
	d.doneCh = nil
	d.state = downloaderIdle
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}
