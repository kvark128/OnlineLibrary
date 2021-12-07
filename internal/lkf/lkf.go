package lkf

import (
	"fmt"
	"io"

	"github.com/kvark128/lkf"
)

type Reader struct {
	src       io.Reader
	buf       []byte
	bufLength int
	c         *lkf.Cryptor
	lastError error
}

func NewReader(src io.Reader) *Reader {
	return &Reader{
		src: src,
		buf: make([]byte, lkf.BlockSize*32), // 16 Kb
		c:   new(lkf.Cryptor),
	}
}

func (r *Reader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	if r.bufLength == 0 {
		for r.bufLength < len(r.buf) && r.lastError == nil {
			var nRead int
			nRead, r.lastError = r.src.Read(r.buf[r.bufLength:])
			r.bufLength += nRead
		}

		if r.bufLength == 0 && r.lastError != nil {
			return 0, r.lastError
		}

		t := r.buf[:r.bufLength]
		r.c.Decrypt(t, t)
	}

	n := copy(p, r.buf[:r.bufLength])
	copy(r.buf, r.buf[n:r.bufLength])
	r.bufLength -= n
	return n, nil
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
	r.lastError = nil

	blockOffset := offset % lkf.BlockSize
	pos, err := seeker.Seek(offset-blockOffset, whence)
	if err != nil {
		return 0, err
	}

	if blockOffset == 0 {
		return pos, nil
	}

	n, err := io.ReadFull(r, make([]byte, blockOffset))
	return pos + int64(n), err
}
