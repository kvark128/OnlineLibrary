package connect

import (
	"fmt"
	"context"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type Connection struct {
	sync.Mutex
	url           string
	client        http.Client
	resp          *http.Response
	closed        bool
	timer *time.Timer
	reads         int64
	contentLength int64
}

func NewConnection(url string) (io.ReadCloser, error) {
	c := &Connection{
		url:    url,
		client: http.Client{},
	}

	if err := c.reset(); err != nil {
		return nil, err
	}

	c.contentLength = c.resp.ContentLength
	return c, nil
}

func (c *Connection) reset() error {
	c.Lock()
	defer c.Unlock()

	if c.closed {
		return fmt.Errorf("connection: closed on the client side")
	}

	ctx, cancelFunc := context.WithCancel(context.TODO())
	c.timer = time.AfterFunc(time.Second * 3, cancelFunc)
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
	c.timer.Reset(time.Second * 3)
	defer c.timer.Stop()

	n, err := c.resp.Body.Read(p)
	c.reads += int64(n)

	if err != nil && c.reads < c.contentLength {
		log.Printf("connection: %v", err)
		for attempt := 1; attempt <= 3; attempt++ {
			log.Printf("Connection recovery: attempt %d/3: reads %d/%d bytes", attempt, c.reads, c.contentLength)
			if c.reset() == nil {
				c.timer.Reset(time.Second * 3)
				n2, err := c.resp.Body.Read(p[n:])
				c.reads += int64(n2)
				return n + n2, err
			}
		}
	}

	return n, err
}

func (c *Connection) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = c.reads + offset
	case io.SeekEnd:
		newOffset = c.contentLength + offset
	default:
		panic(fmt.Sprintf("Invalid whence: %v", whence))
	}

	if newOffset < 0 || newOffset > c.contentLength {
		return 0, fmt.Errorf("connection: invalid offset")
	}

	c.reads = newOffset
	if err := c.reset(); err != nil {
		return 0, err
	}

	return c.reads, nil
}

func (c *Connection) Close() error {
	c.Lock()
	defer c.Unlock()
	c.closed = true
	return c.resp.Body.Close()
}
