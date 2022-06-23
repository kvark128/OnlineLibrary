package buffer

import (
	"bytes"
	"fmt"
	"io"
)

const default_buffer_size = 1024 * 16

type Reader struct {
	source           io.ReadSeeker
	nReadsFromSource int64
	buffer           []byte
	br               *bytes.Reader
	lastErr          error
}

func NewReader(src io.ReadSeeker) *Reader {
	return NewReaderSize(src, default_buffer_size)
}

func NewReaderSize(src io.ReadSeeker, bufSize int) *Reader {
	r := new(Reader)
	r.source = src
	r.buffer = make([]byte, bufSize)
	r.br = bytes.NewReader(nil)
	return r
}

func (r *Reader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, io.ErrShortBuffer
	}
	if r.br.Len() == 0 {
		if r.lastErr != nil {
			return 0, r.lastErr
		}
		var n int
		n, r.lastErr = r.source.Read(r.buffer)
		if n == 0 {
			return 0, r.lastErr
		}
		r.nReadsFromSource += int64(n)
		r.br.Reset(r.buffer[:n])
	}
	return r.br.Read(p)
}

func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	if offset < 0 {
		return 0, fmt.Errorf("negative offset not supported")
	}
	if whence != io.SeekStart {
		return 0, fmt.Errorf("whence %v not supported", whence)
	}
	startBufOffset := r.nReadsFromSource - int64(r.br.Len())
	if offset >= startBufOffset && offset <= r.nReadsFromSource {
		bufOffset := offset - startBufOffset
		r.nReadsFromSource = offset
		if _, err := r.br.Seek(bufOffset, io.SeekStart); err != nil {
			panic(err)
		}
		return offset, nil
	}
	r.br.Reset(nil)
	sourceOffset, err := r.source.Seek(offset, io.SeekStart)
	if err != nil {
		r.lastErr = err
	}
	r.nReadsFromSource = sourceOffset
	return sourceOffset, err
}
