package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("usage: sim0086 <file>")
	}

	b, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("could not read file %s: %v", os.Args[1], err)
	}

	r := newReader((b))

	disassemble(r)
}

func disassemble(r *Reader) {
	var output []string

	output = append(output, "bits 16")

	for !r.isEmpty() {
		b := r.mustPeek()

		// mov
		if (b >> 2) == 0b00100010 {
			mov := parseMov(r)
			if mov == nil {
				log.Fatal("could not parse mov")
			}

			output = append(output, mov.String())
			continue
		}
	}

	fmt.Println(strings.Join(output, "\n"))
}

type MovMode int

const (
	MemoryNoDisplacement MovMode = 0
	Memory8Bit                   = 1
	Memory16Bit                  = 2
	Register                     = 3
)

type Mov struct {
	mode      MovMode
	width     byte
	destReg   byte
	sourceReg byte
}

var REGISTERS_8 = []string{"AL", "CL", "DL", "BL", "AH", "CH", "DH", "BH"}
var REGISTERS_16 = []string{"AX", "CX", "DX", "BX", "SP", "BP", "SI", "DI"}

func (m *Mov) String() string {
	regNames := REGISTERS_16
	if m.width == 0 {
		regNames = REGISTERS_8
	}

	sourceName := regNames[m.sourceReg]
	destName := regNames[m.destReg]

	return fmt.Sprintf("mov %s, %s", destName, sourceName)
}

// 100010dw
// only handles register to register
func parseMov(r *Reader) *Mov {
	firstByte := r.mustRead()
	d := (firstByte >> 1) & 1
	w := firstByte & 1

	secondByte := r.mustRead()
	mod := (secondByte >> 6) & 0b11
	reg := (secondByte >> 3) & 0b111
	rm := secondByte & 0b111

	// only handles register to register
	if mod != 0b11 {
		return nil
	}

	source := reg
	dest := rm

	if d == 1 {
		source = rm
		dest = reg
	}

	result := &Mov{
		mode:      Register,
		width:     w,
		sourceReg: source,
		destReg:   dest,
	}

	return result
}

var ErrOutOfBound = errors.New("out of bound")

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
func (r *Reader) mustPeek() byte {
	result := r.mustRead()
	r.idx--
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

func assert(val bool, msg string) {
	if !val {
		log.Fatal(msg)
	}
}
