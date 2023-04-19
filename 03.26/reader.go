package main

type Reader struct {
	idx int
	buf []byte
}

func newReader(buf []byte) *Reader {
	return &Reader{0, buf}
}
func (r *Reader) isEmpty() bool {
	return r.idx >= len(r.buf)
}
func (r *Reader) read() (byte, error) {
	if r.idx >= len(r.buf) {
		return 0, ErrOutOfBound
	}

	result := r.buf[r.idx]
	r.idx += 1
	return result, nil
}
func (r *Reader) mustPeek(n int) []byte {
	result := r.mustReadN(n)
	r.idx -= n
	return result
}
func (r *Reader) mustRead() byte {
	result, err := r.read()
	if err != nil {
		panic(err)
	}
	return result
}
func (r *Reader) readN(n int) ([]byte, error) {
	if r.idx+n-1 >= len(r.buf) {
		return nil, ErrOutOfBound
	}

	result := r.buf[r.idx : r.idx+n]
	r.idx += n
	return result, nil
}
func (r *Reader) mustReadN(n int) []byte {
	result, err := r.readN(n)
	if err != nil {
		panic(err)
	}
	return result
}

// little endian
func (r *Reader) mustReadUint16() uint16 {
	buf, err := r.readN(2)
	if err != nil {
		panic(err)
	}
	low := uint16(buf[0])
	high := uint16(buf[1])
	return (high << 8) | low
}

func (r *Reader) mustReadUint16W(wide bool) uint16 {
	if wide {
		return r.mustReadUint16()
	}

	return uint16(r.mustRead())
}

func (r *Reader) mustReadInt16() int16 {
	result := r.mustReadUint16()
	return int16(result)
}
func (r *Reader) mustReadInt8() int8 {
	result := r.mustRead()
	return int8(result)
}

// func (r *Reader) mustRead

func (r *Reader) remaining() int {
	result := len(r.buf) - r.idx

	if result < 0 {
		return 0
	}

	return result
}
