package connect

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	buf_size = 1024 * 64
	timeout  = time.Second * 3
)

type Connection struct {
	sync.Mutex
	url                 string
	client              http.Client
	resp                *http.Response
	closed              bool
	buf                 []byte
	bufStart, bufFinish int
	lastErr             error
	timer               *time.Timer
	reads               int64
	contentLength       int64
}

func NewConnection(url string) (io.ReadCloser, error) {
	c := &Connection{
		url: url,
		buf: make([]byte, buf_size),
	}

	if err := c.setNewResponse(); err != nil {
		return nil, err
	}

	if c.resp.ContentLength < 0 {
		return nil, fmt.Errorf("connection: content length is %v", c.resp.ContentLength)
	}

	c.contentLength = c.resp.ContentLength
	return c, nil
}

func (c *Connection) setNewResponse() error {
	c.Lock()
	defer c.Unlock()

	if c.closed {
		return fmt.Errorf("connection: closed on the client side")
	}

	ctx, cancelFunc := context.WithCancel(context.TODO())
	c.timer = time.AfterFunc(timeout, cancelFunc)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", c.reads))

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("connection: %v", err)
	}
	c.timer.Stop()

	if resp.StatusCode != http.StatusPartialContent {
		resp.Body.Close()
		return fmt.Errorf("connection: http code: %d", resp.StatusCode)
	}

	if c.resp != nil {
		c.resp.Body.Close()
	}

	c.resp = resp
	return nil
}

func (c *Connection) Read(p []byte) (int, error) {
	var n, n2 int

	for {
		n2 = copy(p[n:], c.buf[c.bufStart:c.bufFinish])
		n += n2
		if n == len(p) {
			// The p is full
			c.bufStart += n2
			return n, nil
		}

		if c.lastErr != nil {
			return n, c.lastErr
		}

		// The c.buf is empty here. We can fill it
		c.fillBuf()
	}
}

func (c *Connection) fillBuf() {
	var n int
	c.bufStart = 0
	c.bufFinish = 0

	for c.bufFinish < len(c.buf) {
		c.timer.Reset(timeout)
		n, c.lastErr = c.resp.Body.Read(c.buf[c.bufFinish:])
		c.timer.Stop()
		c.bufFinish += n
		c.reads += int64(n)

		if c.lastErr != nil {
			break
		}
	}
}

func (c *Connection) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64

	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = c.reads - int64(c.bufFinish-c.bufStart) + offset
	case io.SeekEnd:
		newOffset = c.contentLength + offset
	default:
		panic(fmt.Sprintf("Invalid whence: %v", whence))
	}

	if newOffset < 0 || newOffset > c.contentLength {
		return 0, fmt.Errorf("connection: invalid offset")
	}

	leftEdge := c.reads - int64(c.bufFinish)
	rightEdge := c.reads

	log.Printf("Seek: newOffset: %v, left: %v, right: %v, start: %v, finish: %v", newOffset, leftEdge, rightEdge, c.bufStart, c.bufFinish)
	if newOffset < leftEdge || newOffset > rightEdge {
		c.reads = newOffset
		c.bufStart = 0
		c.bufFinish = 0
		if err := c.setNewResponse(); err != nil {
			return 0, err
		}
		return c.reads, nil
	}

	c.bufStart = int(newOffset - leftEdge)
	return newOffset, nil
}

func (c *Connection) Close() error {
	c.Lock()
	defer c.Unlock()
	c.closed = true
	c.bufStart = 0
	c.bufFinish = 0
	return c.resp.Body.Close()
}
