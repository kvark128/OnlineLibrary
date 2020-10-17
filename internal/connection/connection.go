package connection

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
	buf_size = 1024 * 16
	timeout  = time.Second * 5
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
	contentType         string
}

func NewConnection(url string) (*Connection, error) {
	c := &Connection{
		url: url,
		buf: make([]byte, buf_size),
	}

	if err := c.createResponse(); err != nil {
		return nil, err
	}

	if c.resp.ContentLength < 0 {
		return nil, fmt.Errorf("connection: content length is %v", c.resp.ContentLength)
	}

	c.contentLength = c.resp.ContentLength
	c.contentType = c.resp.Header.Get("Content-Type")
	return c, nil
}

func (c *Connection) createResponse() error {
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

fillLoop:
	for c.bufFinish < len(c.buf) {
		c.timer.Reset(timeout)
		n, c.lastErr = c.resp.Body.Read(c.buf[c.bufFinish:])
		c.timer.Stop()
		c.bufFinish += n
		c.reads += int64(n)

		if c.lastErr != nil {
			if c.reads < c.contentLength {
				for attempt := 1; attempt <= 3; attempt++ {
					log.Printf("connection recovery: attempt %v/3: reason: %v", attempt, c.lastErr)
					if c.createResponse() == nil {
						c.lastErr = nil
						continue fillLoop
					}
				}
			}
			break fillLoop
		}
	}
}

func (c *Connection) ContentType() string {
	return c.contentType
}

func (c *Connection) Seek(offset int64, whence int) (int64, error) {
	var position int64

	switch whence {
	case io.SeekStart:
		position = offset
	case io.SeekCurrent:
		position = c.reads - int64(c.bufFinish-c.bufStart) + offset
	case io.SeekEnd:
		position = c.contentLength + offset
	default:
		panic(fmt.Sprintf("Invalid whence: %v", whence))
	}

	if position > c.contentLength {
		return 0, fmt.Errorf("connection: offset greater than the end of the body")
	}

	if position < 0 {
		position = 0
	}

	leftEdge := c.reads - int64(c.bufFinish)
	rightEdge := c.reads

	if position >= leftEdge && position <= rightEdge {
		// New position inside the buffer
		c.bufStart = int(position - leftEdge)
		return position, nil
	}

	c.reads = position
	c.bufStart = 0
	c.bufFinish = 0
	if err := c.createResponse(); err != nil {
		return 0, err
	}
	return position, nil
}

func (c *Connection) Close() error {
	c.Lock()
	defer c.Unlock()
	c.closed = true
	c.bufStart = 0
	c.bufFinish = 0
	return c.resp.Body.Close()
}
