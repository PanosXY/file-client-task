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
	"go.uber.org/atomic"
)

// A concurrent downloader

var (
	emptyUrlErr       = errors.New("The subscribed url is empty")
	emptyQueueErr     = errors.New("The subscribed filenames queue is empty")
	nilRespHandlerErr = errors.New("The subscribed response handler function is nil")
	alreadyStartedErr = errors.New("Is already started")
	notSubedYetErr    = errors.New("Not subscribed yet")
)

type downloaderState int

const (
	downloaderIdle downloaderState = iota
	downloaderStarted
)

type handlerFunc func(string, io.ReadCloser) error

type ConcurrentDownloader struct {
	mutex       sync.Mutex
	wg          sync.WaitGroup
	routinesN   atomic.Uint32
	url         string
	queue       []string
	state       downloaderState
	workers     uint32
	respHandler handlerFunc
	log         *utils.Logger
}

func NewConcurrentDownloader(log *utils.Logger, workers uint) *ConcurrentDownloader {
	d := new(ConcurrentDownloader)
	d.log = log
	d.workers = uint32(workers)
	d.state = downloaderIdle
	return d
}

func (d *ConcurrentDownloader) Subscribe(url string, queue []string, rh handlerFunc) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if url == "" {
		return emptyUrlErr
	}
	d.url = url
	if len(queue) == 0 {
		return emptyQueueErr
	}
	d.queue = queue
	if rh == nil {
		return nilRespHandlerErr
	}
	d.respHandler = rh
	return nil
}

func (d *ConcurrentDownloader) Start() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.state == downloaderStarted {
		return alreadyStartedErr
	}
	if len(d.queue) == 0 {
		return notSubedYetErr
	}
	d.state = downloaderStarted
	idx := 0
	maxWorkers := min(uint32(len(d.queue)), d.workers)
	for {
		if d.routinesN.Load() <= maxWorkers {
			d.wg.Add(1)
			go d.getFile(d.queue[idx])
			idx++
		}
		if idx == len(d.queue) {
			break
		}
	}
	d.wg.Wait()
	d.cleanup()
	return nil
}

func (d *ConcurrentDownloader) getFile(filename string) {
	d.routinesN.Inc()
	defer d.routinesN.Dec()
	defer d.wg.Done()
	req, err := http.NewRequest(http.MethodGet, d.url+"/"+filename, nil)
	if err != nil {
		d.log.Error(fmt.Sprintf("Couldn't download file '%s': %v", filename, err))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	if err := d.respHandler(filename, resp.Body); err != nil {
		d.log.Error(fmt.Sprintf("Failed operate in the file '%s': %v", filename, err))
		return
	}
}

func (d *ConcurrentDownloader) cleanup() {
	d.url = ""
	d.queue = []string{}
	d.state = downloaderIdle
}

func min(x, y uint32) uint32 {
	if x > y {
		return y
	}
	return x
}
