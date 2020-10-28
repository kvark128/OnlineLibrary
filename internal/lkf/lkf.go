package lkf

import (
	"fmt"
	"io"

	"github.com/kvark128/lkf"
)

type Reader struct {
	src       io.ReadCloser
	buf       []byte
	bufLength int
	c         *lkf.Cryptor
	lastErr   error
}

func NewReader(src io.ReadCloser) *Reader {
	return &Reader{
		src: src,
		buf: make([]byte, lkf.BlockSize*64),
		c:   new(lkf.Cryptor),
	}
}

func (r *Reader) Read(p []byte) (int, error) {
	var n, n2 int
	for {
		n2 = copy(p[n:], r.buf[:r.bufLength])
		n += n2
		if n == len(p) {
			r.bufLength = copy(r.buf, r.buf[n2:r.bufLength])
			return n, nil
		}

		if r.lastErr != nil {
			return n, r.lastErr
		}

		r.bufLength, r.lastErr = r.src.Read(r.buf)
		t := r.buf[:r.bufLength]
		r.c.Decrypt(t, t)
	}
}

func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	seeker, ok := r.src.(io.Seeker)
	if !ok {
		panic("LKFReader: r.src is not seeker")
	}

	if whence != io.SeekStart {
		return 0, fmt.Errorf("LKFReader: Seek: only io.SeekStart is supported")
	}

	r.bufLength = 0
	r.lastErr = nil

	blockOffset := offset % lkf.BlockSize
	pos, err := seeker.Seek(offset-blockOffset, whence)
	if err != nil {
		return 0, err
	}

	if blockOffset == 0 {
		return pos, nil
	}

	n, err := r.Read(make([]byte, blockOffset))
	return pos + int64(n), err
}

func (r *Reader) Close() error {
	r.bufLength = 0
	return r.src.Close()
}
