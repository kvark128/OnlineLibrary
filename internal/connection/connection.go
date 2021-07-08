package connection

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/log"
)

const (
	buf_size = 1024 * 256
)

type Connection struct {
	url           string
	client        http.Client
	ctx           context.Context
	resp          *http.Response
	buf           []byte
	rBuf, wBuf    int
	lastErr       error
	timer         *time.Timer
	reads         int64
	contentLength int64
}

func NewConnection(url string) (*Connection, error) {
	return NewConnectionWithContext(context.TODO(), url)
}

func NewConnectionWithContext(ctx context.Context, url string) (*Connection, error) {
	c := &Connection{
		url: url,
		buf: make([]byte, buf_size),
		ctx: ctx,
	}

	if err := c.createResponse(1); err != nil {
		return nil, err
	}

	if c.resp.ContentLength <= 0 {
		return nil, fmt.Errorf("content length <= 0")
	}

	c.contentLength = c.resp.ContentLength
	return c, nil
}

func (c *Connection) createResponse(nAttempts int) error {
	ctx, cancelFunc := context.WithCancel(c.ctx)
	c.timer = time.AfterFunc(config.HTTPTimeout, cancelFunc)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", c.reads))

	var resp *http.Response
	var err error
	for nAttempts > 0 {
		nAttempts--
		resp, err = c.client.Do(req)
		if err == nil {
			break
		}
	}
	c.timer.Stop()

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusPartialContent {
		resp.Body.Close()
		return fmt.Errorf("unexpected http status code: %d", resp.StatusCode)
	}

	if c.resp != nil {
		c.resp.Body.Close()
	}

	c.resp = resp
	return nil
}

func (c *Connection) Read(p []byte) (int, error) {
	if c.rBuf == c.wBuf {
		if c.lastErr != nil {
			return 0, c.lastErr
		}
		c.fillBuf()
	}

	n := copy(p, c.buf[c.rBuf:c.wBuf])
	c.rBuf += n
	if c.rBuf == c.wBuf {
		return n, c.lastErr
	}
	return n, nil
}

func (c *Connection) fillBuf() {
	if c.rBuf != c.wBuf {
		panic("c.buf contains unread data")
	}

	chunkSize := 1024 * 8
	if len(c.buf)-c.rBuf < chunkSize {
		c.resetBuf()
	}

	chunk := c.buf[c.wBuf : c.wBuf+chunkSize]
	for {
		c.timer.Reset(config.HTTPTimeout)
		n, err := c.resp.Body.Read(chunk)
		c.timer.Stop()
		c.wBuf += n
		c.reads += int64(n)

		if err != nil && err != context.Canceled && c.reads < c.contentLength {
			log.Warning("connection recovery: %v", err)
			if c.createResponse(3) == nil {
				if n == 0 {
					// We have no read data. Repeat reading
					continue
				}
				// We have read the data. Just ignore the error without re-reading
				err = nil
			}
		}

		c.lastErr = err
		break
	}
}

func (c *Connection) resetBuf() {
	c.rBuf = 0
	c.wBuf = 0
}

func (c *Connection) Seek(offset int64, whence int) (int64, error) {
	var position int64
	switch whence {
	case io.SeekStart:
		position = offset
	case io.SeekCurrent:
		position = c.reads + offset
	case io.SeekEnd:
		position = c.contentLength + offset
	default:
		panic(fmt.Sprintf("Invalid whence: %v", whence))
	}

	if position > c.contentLength {
		return 0, fmt.Errorf("offset greater than the end of the body")
	}

	if position < 0 {
		position = 0
	}

	// Edges of the buffer relative to the beginning of the body
	leftEdge := c.reads - int64(c.wBuf)
	rightEdge := c.reads

	if position >= leftEdge && position <= rightEdge {
		// New position inside the buffer
		c.rBuf = int(position - leftEdge)
		return position, nil
	}

	c.resetBuf()
	if c.lastErr != nil {
		return 0, c.lastErr
	}

	c.reads = position
	if err := c.createResponse(1); err != nil {
		return 0, err
	}
	return position, nil
}

func (c *Connection) Close() error {
	c.resetBuf()
	c.lastErr = fmt.Errorf("connection was closed")
	return c.resp.Body.Close()
}
