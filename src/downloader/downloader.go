package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/PanosXY/file-client-task/utils"
)

// A non blocking concurrent downloader

type downloaderState int

const (
	downloaderIdle downloaderState = iota
	downloaderStarted
)

type handlerFunc func(string, io.ReadCloser) error

type ConcurrentDownloader struct {
	wg          sync.WaitGroup
	mutex       sync.Mutex
	url         string
	queue       []string
	doneCh      chan struct{}
	state       downloaderState
	workers     uint
	respHandler handlerFunc
	log         *utils.Logger
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

func (d *ConcurrentDownloader) Subscribe(url string, queue []string, doneCh chan struct{}, rh handlerFunc) error {
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
	if doneCh == nil {
		return errors.New("The subscribed done channel is nil")
	}
	d.doneCh = doneCh
	// Response Handler is not mandatory
	d.respHandler = rh
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
	if d.respHandler != nil {
		if err := d.respHandler(filename, resp.Body); err != nil {
			d.log.Error(fmt.Sprintf("Failed operate in the file '%s': %v", filename, err))
			return
		}
	}
}

func (d *ConcurrentDownloader) cleanup() {
	d.mutex.Lock()
	d.mutex.Unlock()
	d.url = ""
	d.queue = []string{}
	d.doneCh = nil
	d.state = downloaderIdle
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}
