package lkf

import (
	"io"

	"github.com/kvark128/lkf"
)

type LKFReader struct {
	src       io.Reader
	buf       []byte
	bufLength int
	c         *lkf.Cryptor
	lastErr   error
}

func NewLKFReader(src io.Reader) *LKFReader {
	r := &LKFReader{
		src: src,
		buf: make([]byte, lkf.BlockSize*128),
		c:   new(lkf.Cryptor),
	}
	return r
}

func (r *LKFReader) Read(p []byte) (int, error) {
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

func (r *LKFReader) Seek(offset int64, whence int) (int64, error) {
	seeker, ok := r.src.(io.Seeker)
	if !ok {
		panic("LKFReader: r.src is not seeker")
	}

	r.bufLength = 0
	r.lastErr = nil

	pos, err := seeker.Seek(offset, whence)
	if err != nil {
		return 0, err
	}

	newOffset := pos % lkf.BlockSize
	if newOffset == 0 {
		return pos, nil
	}

	pos, err = seeker.Seek(-newOffset, io.SeekCurrent)
	if err != nil {
		return 0, err
	}

	n, err := r.Read(make([]byte, newOffset))
	return pos + int64(n), err
}
