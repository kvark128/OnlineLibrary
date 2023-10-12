package connection

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/log"
)

var (
	ConnectionWasClosed = errors.New("connection was closed")
	client              = new(http.Client)
)

type Connection struct {
	url           string
	ctx           context.Context
	resp          *http.Response
	lastErr       error
	logger        *log.Logger
	timer         *time.Timer
	reads         int64
	contentLength int64
}

func NewConnection(url string, logger *log.Logger) (*Connection, error) {
	return NewConnectionWithContext(context.TODO(), url, logger)
}

func NewConnectionWithContext(ctx context.Context, url string, logger *log.Logger) (*Connection, error) {
	c := &Connection{
		url:    url,
		ctx:    ctx,
		logger: logger,
	}

	if err := c.createResponse(); err != nil {
		return nil, err
	}

	if c.resp.ContentLength <= 0 {
		return nil, fmt.Errorf("content length <= 0")
	}

	c.contentLength = c.resp.ContentLength
	return c, nil
}

func (c *Connection) createResponse() error {
	ctx, cancelFunc := context.WithCancel(c.ctx)
	c.timer = time.AfterFunc(config.HTTPTimeout, cancelFunc)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", c.reads))

	var resp *http.Response
	for attempt := 0; attempt < 3; attempt++ {
		resp, err = client.Do(req)
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
	if c.resp == nil {
		return 0, ConnectionWasClosed
	}

	c.timer.Reset(config.HTTPTimeout)
	n, err := c.resp.Body.Read(p)
	c.timer.Stop()
	c.reads += int64(n)

	if err != nil && err != context.Canceled && c.reads < c.contentLength {
		c.logger.Warning("Connection recovery: %v", err)
		if c.createResponse() == nil {
			err = nil
		}
	}
	return n, err
}

func (c *Connection) Seek(offset int64, whence int) (int64, error) {
	if c.resp == nil {
		return 0, ConnectionWasClosed
	}

	var pos int64
	switch whence {
	case io.SeekStart:
		pos = offset
	case io.SeekCurrent:
		pos = c.reads + offset
	case io.SeekEnd:
		pos = c.contentLength + offset
	default:
		panic(fmt.Sprintf("Invalid whence: %v", whence))
	}

	if pos > c.contentLength {
		return 0, fmt.Errorf("offset greater than the end of the body")
	}

	if pos < 0 {
		pos = 0
	}

	c.reads = pos
	if err := c.createResponse(); err != nil {
		return 0, err
	}
	return pos, nil
}

func (c *Connection) Close() error {
	if c.resp == nil {
		return ConnectionWasClosed
	}
	err := c.resp.Body.Close()
	c.resp = nil
	return err
}
