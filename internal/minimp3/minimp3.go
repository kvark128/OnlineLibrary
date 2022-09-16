package minimp3

//#define MINIMP3_IMPLEMENTATION
//#include <minimp3.h>
import "C"

import (
	"errors"
	"io"
	"unsafe"
)

const maxSamplesPerFrame = 1152 * 2

// Decoder decode the mp3 stream by minimp3
type Decoder struct {
	source               io.Reader
	mp3, pcm             []byte
	mp3Length, pcmLength int
	lastError            error
	decode               C.mp3dec_t
	info                 C.mp3dec_frame_info_t
}

// NewDecoder creates and returns a new mp3 decoder with the default internal buffer size.
func NewDecoder(source io.Reader) *Decoder {
	d := &Decoder{
		source: source,
		mp3:    make([]byte, 1024*16),
		pcm:    make([]byte, maxSamplesPerFrame*C.sizeof_short),
	}

	C.mp3dec_init(&d.decode)
	return d
}

// Read copies the decoded audio data from the internal buffer to p and returns the number of bytes copied.
// If the internal buffer is empty, then Read first tries to read mp3 data from the source and decode it.
// If len(p) == 0, then Read will return zero bytes, but if the internal buffer is empty, then before that it will still try to decode one frame and fill the internal buffer.
func (d *Decoder) Read(p []byte) (int, error) {
	for d.pcmLength == 0 {
		// If possible, fill the mp3 buffer completely
		for d.mp3Length < len(d.mp3) && d.lastError == nil {
			n, err := d.source.Read(d.mp3[d.mp3Length:])
			d.mp3Length += n
			d.lastError = err
		}

		samples := C.mp3dec_decode_frame(&d.decode,
			(*C.uint8_t)(unsafe.Pointer(&d.mp3[0])), C.int(d.mp3Length),
			(*C.mp3d_sample_t)(unsafe.Pointer(&d.pcm[0])), &d.info,
		)

		if d.info.frame_bytes == 0 {
			return 0, d.lastError
		}

		d.mp3Length = copy(d.mp3, d.mp3[d.info.frame_bytes:d.mp3Length])
		d.pcmLength = int(samples * d.info.channels * C.sizeof_short)
	}

	n := copy(p, d.pcm[:d.pcmLength])
	// If there is any data left in the pcm buffer, then move it to the beginning of the buffer
	copy(d.pcm, d.pcm[n:d.pcmLength])
	d.pcmLength -= n
	return n, nil
}

// Seek sets a new position for reading audio data.
// At least one decoded frame is required to set a new position.
// If there are no decoded frames, then Seek will return an error.
// The mp3 data source must support the io.Seeker interface.
// If the source doesn't support io.Seeker, then Seek will panic.
func (d *Decoder) Seek(offset int64, whence int) (int64, error) {
	seeker, ok := d.source.(io.Seeker)
	if !ok {
		panic("minimp3: d.source is not seeker")
	}

	mp3BytesPerMsec := int64(d.info.bitrate_kbps) / 8
	pcmBytesPerMsec := int64(d.info.channels*C.sizeof_short*d.info.hz) / 1000

	if mp3BytesPerMsec == 0 || pcmBytesPerMsec == 0 {
		// There is no information about audio data. Probably not a single frame has been decoded yet
		return 0, errors.New("no frame available")
	}

	var mp3Offset int64
	if whence == io.SeekCurrent {
		// If seeking is performed from the current position, then be aware of buffered audio data
		offset -= int64(d.pcmLength)
		mp3Offset = offset / pcmBytesPerMsec * mp3BytesPerMsec
		mp3Offset -= int64(d.mp3Length)
	} else {
		// If seeking is performed from the beginning or the end, then the buffered data does not matter
		mp3Offset = offset / pcmBytesPerMsec * mp3BytesPerMsec
	}

	// Internal buffers must always be cleared, regardless of the result of calling the Seek method
	d.mp3Length = 0
	d.pcmLength = 0

	mp3Pos, err := seeker.Seek(mp3Offset, whence)
	if err != nil {
		return 0, err
	}

	pcmPos := mp3Pos / mp3BytesPerMsec * pcmBytesPerMsec
	return pcmPos, nil
}

// SampleRate returns the sample rate of the last decoded frame.
// If no frames have been decoded yet, then it will return 0.
func (d *Decoder) SampleRate() int {
	return int(d.info.hz)
}

// Channels returns the number of channels of the last decoded frame.
// If no frames have been decoded yet, then it will return 0.
func (d *Decoder) Channels() int {
	return int(d.info.channels)
}

// Bitrate returns the mp3 bitrate of the last decoded frame.
// If no frames have been decoded yet, then it will return 0.
func (d *Decoder) Bitrate() int {
	return int(d.info.bitrate_kbps)
}
