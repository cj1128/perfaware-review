package main

import (
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/k0kubun/pp"
	"golang.org/x/exp/constraints"
)

var debug *bool

func main() {
	check := flag.Bool("check", false, "enable check mode")
	debug = flag.Bool("debug", false, "enable debug mode")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Println("usage: ./sim0086 [-check] [-debug] <binary>")
		os.Exit(0)
	}

	file := flag.Arg(0)
	base := filepath.Base(file)

	result, err := disasembleFile(file)

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("=== Result")
	fmt.Println(result)

	// use `nasm` to assemble our disassemble file
	// and then compare it to origin binary
	// source: a
	// our disassemble: a.sim0086.asm
	// nasm reassemble: a.sim0086
	// then compare a and a.sim0086
	if *check {
		tmpPath := fmt.Sprintf("%s.sim0086.asm", base)

		if err := ioutil.WriteFile(tmpPath, []byte(result), 0644); err != nil {
			log.Fatalf("could not write result into %s: %v\n", tmpPath, err)
		}

		if err := nasmAssembleFile(tmpPath); err != nil {
			log.Fatalf("nasm error: %v", err)
		}

		nasmPath := fmt.Sprintf("%s.sim0086", base)

		same, err := compareTwoFiles(file, nasmPath)
		if err != nil {
			log.Fatalf("could not compare %s and %s: %v", file, nasmPath, err)
		}

		if !same {
			fmt.Println("=== Error, not the same")
		} else {
			fmt.Println("=== Ok")
		}
	}
}

func nasmAssembleFile(fp string) error {
	command := "nasm"

	cmd := exec.Command(command, fp)

	_, err := cmd.Output()

	if err != nil {
		return err
	}

	return nil
}

// return true if two files are the same
func compareTwoFiles(f1, f2 string) (bool, error) {
	h1, err := hashFile(f1)
	if err != nil {
		return false, fmt.Errorf("could not hash %s: %v", f1, err)
	}

	h2, err := hashFile(f2)
	if err != nil {
		return false, fmt.Errorf("could not hash %s: %v", f2, err)
	}

	return h1 == h2, nil
}

func disasembleFile(fp string) (string, error) {
	b, err := ioutil.ReadFile(fp)

	if err != nil {
		return "", fmt.Errorf("could not read file: %w", err)
	}

	output, err := disassemble(b)
	if err != nil {
		return "", fmt.Errorf("failed to disassemble: %w", err)
	}

	return strings.Join(output, "\n"), nil
}

func disassemble(buf []byte) ([]string, error) {
	r := newReader(buf)
	var output []string

	output = append(output, "bits 16")

	for !r.isEmpty() {
		bs := r.mustPeek(min(4, r.remaining()))
		firstByte := bs[0]

		switch true {
		case isMov(firstByte):
			{
				mov := parseMov(r)

				if mov == nil {
					return nil, fmt.Errorf("invalid mov instruction: %#v", bs)
				}

				if *debug {
					pp.Println(mov)
				}
				output = append(output, mov.String())

			}
		default:
			{
				return nil, fmt.Errorf("unknown instruction: %#v", bs)
			}
		}
	}

	return output, nil
}

func isMov(b byte) bool {
	return (b>>2) == 0b100010 || (b>>1) == 0b1100011 || (b>>4) == 0b1011 || (b>>1) == 0b1010000 || (b>>1) == 0b1010001 || b == 0b10001110 || b == 0b10001100
}

type MovType int

const (
	RegisteryOrMemory_ToOrFrom_Register MovType = iota
	Immediate_To_Register
	Immediate_To_RegisterOrMemory
	Memory_To_Accumulator
	Accumulator_To_Memory
)

type Mov struct {
	typ        MovType
	d          byte
	w          byte
	mod        byte
	reg        byte
	rm         byte
	data       uint16
	disp       int16
	dispLength byte
}

var REGISTERS_8 = []string{"al", "cl", "dl", "bl", "ah", "ch", "dh", "bh"}
var REGISTERS_16 = []string{"ax", "cx", "dx", "bx", "sp", "bp", "si", "di"}

func (m *Mov) regName() string {
	regNames := REGISTERS_16
	if m.w == 0 {
		regNames = REGISTERS_8
	}

	return regNames[m.reg]
}
func (m *Mov) rmName() string {
	switch m.mod {
	case 0b11:
		{
			regNames := REGISTERS_16

			if m.w == 0 {
				regNames = REGISTERS_8
			}

			return regNames[m.rm]
		}
	case 0b00:
		fallthrough
	case 0b01:
		fallthrough
	case 0b10:
		{
			base := []string{"bx + si", "bx + di", "bp + si", "bp + di", "si", "di", "bp", "bx"}[m.rm]

			if m.mod == 0b00 {
				if m.rm == 0b110 {
					return fmt.Sprintf("[%d]", uint16(m.disp))
				}
				return fmt.Sprintf("[%s]", base)
			}

			if m.disp < 0 {
				return fmt.Sprintf("[%s - %d]", base, -m.disp)
			}

			return fmt.Sprintf("[%s + %d]", base, m.disp)
		}
	}
	pp.Println(m)
	panic("unreachable")
}

func (m *Mov) String() string {
	switch m.typ {
	case RegisteryOrMemory_ToOrFrom_Register:
		{
			source := m.regName()
			target := m.rmName()

			if m.d == 1 {
				t := source
				source = target
				target = t
			}

			return fmt.Sprintf("mov %s, %s", target, source)

		}
	case Immediate_To_Register:
		{
			return fmt.Sprintf("mov %s, %d", m.regName(), m.data)
		}
	case Immediate_To_RegisterOrMemory:
		{
			// register
			if m.mod == 0b11 {
				return fmt.Sprintf("mov %s, %d", m.rmName(), m.data)
			}

			dataType := "byte"
			if m.dispLength == 2 {
				dataType = "word"
			}
			return fmt.Sprintf("mov %s, %s %d", m.rmName(), dataType, m.data)
		}
	case Memory_To_Accumulator:
		{
			return fmt.Sprintf("mov ax, [%d]", m.data)
		}
	case Accumulator_To_Memory:
		{
			return fmt.Sprintf("mov [%d], ax", m.data)
		}
	default:
		{
			panic("TODO: Mov String() method")
		}
	}
}

// 100010dw
// only handles register to register
func parseMov(r *Reader) *Mov {
	firstByte := r.mustRead()

	result := &Mov{}

	switch true {
	// register/memory to/from register
	case (firstByte >> 2) == 0b100010:
		{
			result.typ = RegisteryOrMemory_ToOrFrom_Register
			result.d = (firstByte >> 1) & 1
			result.w = firstByte & 1

			secondByte := r.mustRead()

			result.mod = (secondByte >> 6) & 0b11
			result.reg = (secondByte >> 3) & 0b111
			result.rm = secondByte & 0b111

			if result.mod == 0b01 {
				result.disp = int16(int8(r.mustRead()))
				result.dispLength = 1
			} else if result.mod == 0b10 || (result.mod == 0b00 && result.rm == 0b110) {
				result.disp = r.mustReadInt16()
				result.dispLength = 2
			}
		}
	// immediate to register
	case (firstByte >> 4) == 0b1011:
		{
			result.typ = Immediate_To_Register
			result.w = (firstByte >> 3) & 1
			result.reg = firstByte & 0b111

			if result.w == 1 {
				result.data = r.mustReadUint16()
			} else {
				result.data = uint16(r.mustRead())
			}
		}
	// immediate to register/memory
	case (firstByte >> 1) == 0b1100011:
		{
			result.typ = Immediate_To_RegisterOrMemory
			result.w = firstByte & 1

			secondByte := r.mustRead()
			result.mod = (secondByte >> 6) & 0b11
			result.rm = secondByte & 0b111

			if (result.mod == 0b00 && result.rm == 0b110) || result.mod == 0b10 {
				result.disp = r.mustReadInt16()
				result.dispLength = 2
			} else if result.mod == 0b01 {
				result.disp = int16(int8(r.mustRead()))
				result.dispLength = 1
			}

			if result.w == 1 {
				result.data = r.mustReadUint16()
			} else {
				result.data = uint16(r.mustRead())
			}
		}
	// memory to accumulator
	case (firstByte >> 1) == 0b1010000:
		{
			result.typ = Memory_To_Accumulator
			result.data = r.mustReadUint16()
		}
	// memory to accumulator
	case (firstByte >> 1) == 0b1010001:
		{
			result.typ = Accumulator_To_Memory
			result.data = r.mustReadUint16()
		}
	default:
		{
			log.Fatalf("invalid first byte of MOV: %b\n", firstByte)
		}
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
func (r *Reader) mustReadInt16() int16 {
	result := r.mustReadUint16()
	return int16(result)
}
func (r *Reader) remaining() int {
	result := len(r.buf) - r.idx

	if result < 0 {
		return 0
	}

	return result
}

func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()

	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	hashValue := hash.Sum(nil)

	return fmt.Sprintf("%x", hashValue), nil
}
