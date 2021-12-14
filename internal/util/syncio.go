package util

import (
	"io"
	"sync"
)

type ReadSeeker struct {
	*Reader
	*Seeker
}

func NewReadSeeker(rs io.ReadSeeker, l sync.Locker) *ReadSeeker {
	return &ReadSeeker{Reader: NewReader(rs, l), Seeker: NewSeeker(rs, l)}
}

type ReadWriter struct {
	*Reader
	*Writer
}

func NewReadWriter(rw io.ReadWriter, l sync.Locker) *ReadWriter {
	return &ReadWriter{Reader: NewReader(rw, l), Writer: NewWriter(rw, l)}
}

type Reader struct {
	locker sync.Locker
	reader io.Reader
}

func NewReader(r io.Reader, l sync.Locker) *Reader {
	return &Reader{locker: l, reader: r}
}

func (r *Reader) Read(p []byte) (int, error) {
	r.locker.Lock()
	defer r.locker.Unlock()
	return r.reader.Read(p)
}

type Writer struct {
	locker sync.Locker
	writer io.Writer
}

func NewWriter(w io.Writer, l sync.Locker) *Writer {
	return &Writer{locker: l, writer: w}
}

func (w *Writer) Write(p []byte) (int, error) {
	w.locker.Lock()
	defer w.locker.Unlock()
	return w.writer.Write(p)
}

type Seeker struct {
	locker sync.Locker
	seeker io.Seeker
}

func NewSeeker(s io.Seeker, l sync.Locker) *Seeker {
	return &Seeker{locker: l, seeker: s}
}

func (s *Seeker) Seek(offset int64, whence int) (int64, error) {
	s.locker.Lock()
	defer s.locker.Unlock()
	return s.seeker.Seek(offset, whence)
}
