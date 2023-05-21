package main

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/k0kubun/pp"
)

func disasembleFile(fp string) ([]Command, error) {
	b, err := ioutil.ReadFile(fp)

	if err != nil {
		return nil, fmt.Errorf("could not read file: %w", err)
	}

	cmds, err := disassemble(b)
	if err != nil {
		return nil, fmt.Errorf("failed to disassemble: %w", err)
	}

	return cmds, nil
}

func disassemble(buf []byte) ([]Command, error) {
	var cmds []Command

	r := newReader(buf)

	for !r.isEmpty() {
		bs := r.mustPeek(min(4, r.remaining()))
		firstByte := bs[0]

		var cmd Command

		switch true {
		case isMov(firstByte):
			{
				mov := parseMov(r)

				if mov.typ == Mov_Invalid {
					return nil, fmt.Errorf("invalid mov instruction: %#v", bs)
				}

				cmd = &mov
			}
		case isArithmetic(firstByte):
			{
				add := parseArithmetic(r)

				if add.typ == Arithmetic_Invalid {
					return nil, fmt.Errorf("invalid add instruction: %#v", bs)
				}
				cmd = &add
			}
		case isJumpOrLoop(firstByte):
			{
				j := JumpOrLoop{}
				j.op = r.mustRead()
				j.inc = r.mustReadInt8()

				cmd = &j
			}
		default:
			{
				return nil, fmt.Errorf("unknown instruction: %#v", bs)
			}
		}

		if *debugFlag {
			pp.Println(cmd)
		}

		cmds = append(cmds, cmd)
	}

	return cmds, nil
}

func isMov(b byte) bool {
	return (b>>2) == 0b100010 || (b>>1) == 0b1100011 || (b>>4) == 0b1011 || (b>>1) == 0b1010000 || (b>>1) == 0b1010001 || b == 0b10001110 || b == 0b10001100
}

func isArithmetic(b byte) bool {
	return isAdd(b) || isSub(b) || isCmp(b) ||
		(b>>2) == 0b100000 // common
}
func isAdd(b byte) bool {
	return (b>>2) == 0b0 || (b>>1) == 0b10
}
func isSub(b byte) bool {
	return (b>>2) == 0b1010 || (b>>1) == 0b10110
}
func isCmp(b byte) bool {
	return (b>>2) == 0b1110 || (b>>1) == 0b11110
}

func isJumpOrLoop(b byte) bool {
	return (b>>4) == 0b0111 || (b>>2) == 0b111000
}

type MovType int

const (
	Mov_Invalid MovType = iota
	Mov_RegisteryOrMemory_ToOrFrom_Register
	Mov_Immediate_To_Register
	Mov_Immediate_To_RegisterOrMemory
	Mov_Memory_To_Accumulator
	Mov_Accumulator_To_Memory
)

type Command interface {
	Disassemble() string
}

type Mov struct {
	typ MovType
	d   byte
	w   byte

	Common
	data uint16
}
type Common struct {
	mod  byte
	reg  byte
	rm   byte
	disp int16
}

type ArithmeticType int

const (
	Arithmetic_Invalid ArithmeticType = iota
	Arithmetic_RegOrMemory_With_Register_To_Either
	Arithmetic_Immediate_To_RegisterOrMemory
	Arithmetic_Immediate_To_Accumulator
)

type ArithmeticOp int

const (
	Arithmetic_Add ArithmeticOp = iota
	Arithmetic_Sub
	Arithmetic_Cmp
)

type Arithmetic struct {
	typ ArithmeticType
	op  ArithmeticOp
	Common
	data      uint16
	d         byte
	s         byte
	w         byte
	firstByte byte
}

type JumpOrLoop struct {
	op  uint8
	inc int8
}

// starts with '0b0111'
var Jump_Labels = []string{
	"jo",
	"jno",
	"jb",
	"jnb",
	"jz",
	"jnz",
	"jbe",
	"jnbe",
	"js",
	"jns",
	"jp",
	"jnp",
	"jl",
	"jnl",
	"jle",
	"jnle",
}

// starts with '0b11100'
var Loop_Lables = []string{
	"loopnz",
	"loopz",
	"loop",
	"jcxz",
}

func (j *JumpOrLoop) Disassemble() string {
	return fmt.Sprintf("%s $+2%+d", j.opName(), j.inc)

}
func (j *JumpOrLoop) opName() string {
	if (j.op >> 4) == 0b0111 {
		return Jump_Labels[j.op&0b1111]
	}

	return Loop_Lables[j.op&0b11]
}

func (a *Arithmetic) opName() string {
	b := a.firstByte
	switch true {
	case isAdd(b):
		{
			return "add"
		}
	case isSub(b):
		{
			return "sub"
		}
	case isCmp(b):
		{
			return "cmp"
		}
	default:
		{
			switch a.reg {
			case 0b000:
				{
					return "add"
				}
			case 0b111:
				{
					return "cmp"
				}
			case 0b101:
				{
					return "sub"
				}
			}
		}
	}

	panic("unreachable")
}

func (a *Arithmetic) Disassemble() string {
	op := a.opName()

	switch a.typ {
	case Arithmetic_RegOrMemory_With_Register_To_Either:
		{
			source := a.regName(a.w)
			target := a.rmName(a.w)

			if a.d == 1 {
				t := source
				source = target
				target = t
			}

			return fmt.Sprintf("%s %s, %s", op, target, source)
		}
	case Arithmetic_Immediate_To_RegisterOrMemory:
		{
			// register
			if a.mod == 0b11 {
				return fmt.Sprintf("%s %s, %d", op, a.rmName(a.w), a.data)
			}

			dataType := "byte"
			if a.w == 1 {
				dataType = "word"
			}

			return fmt.Sprintf("%s %s %s, %d", op, dataType, a.rmName(a.w), a.data)
		}

	case Arithmetic_Immediate_To_Accumulator:
		{
			regName := "ax"
			if a.w == 0 {
				regName = "al"
			}
			return fmt.Sprintf("%s %s, %d", op, regName, a.data)
		}
	}

	panic("unreachable")
}

var REGISTERS_8 = []string{"al", "cl", "dl", "bl", "ah", "ch", "dh", "bh"}
var REGISTERS_16 = []string{"ax", "cx", "dx", "bx", "sp", "bp", "si", "di"}

func regName(idx byte, wide byte) string {
	regNames := REGISTERS_16
	if wide == 0 {
		regNames = REGISTERS_8
	}

	return regNames[idx]
}

func (c *Common) regName(wide byte) string {
	return regName(c.reg, wide)
}
func (c *Common) rmName(wide byte) string {
	switch c.mod {
	case 0b11:
		{
			regNames := REGISTERS_16

			if wide == 0 {
				regNames = REGISTERS_8
			}

			return regNames[c.rm]
		}
	case 0b00:
		fallthrough
	case 0b01:
		fallthrough
	case 0b10:
		{
			base := []string{"bx + si", "bx + di", "bp + si", "bp + di", "si", "di", "bp", "bx"}[c.rm]

			if c.mod == 0b00 {
				if c.rm == 0b110 {
					return fmt.Sprintf("[%d]", uint16(c.disp))
				}
				return fmt.Sprintf("[%s]", base)
			}

			if c.disp < 0 {
				return fmt.Sprintf("[%s - %d]", base, -c.disp)
			}

			if c.disp == 0 {
				return fmt.Sprintf("[%s]", base)
			}

			return fmt.Sprintf("[%s + %d]", base, c.disp)
		}
	}

	panic("unreachable")
}

func (m *Mov) Disassemble() string {
	switch m.typ {
	case Mov_RegisteryOrMemory_ToOrFrom_Register:
		{
			source := m.regName(m.w)
			target := m.rmName(m.w)

			if m.d == 1 {
				t := source
				source = target
				target = t
			}

			return fmt.Sprintf("mov %s, %s", target, source)

		}
	case Mov_Immediate_To_Register:
		{
			return fmt.Sprintf("mov %s, %d", m.regName(m.w), m.data)
		}
	case Mov_Immediate_To_RegisterOrMemory:
		{
			// register
			if m.mod == 0b11 {
				return fmt.Sprintf("mov %s, %d", m.rmName(m.w), m.data)
			}

			dataType := "byte"
			if m.w == 1 {
				dataType = "word"
			}
			return fmt.Sprintf("mov %s, %s %d", m.rmName(m.w), dataType, m.data)
		}
	case Mov_Memory_To_Accumulator:
		{
			return fmt.Sprintf("mov ax, [%d]", m.data)
		}
	case Mov_Accumulator_To_Memory:
		{
			return fmt.Sprintf("mov [%d], ax", m.data)
		}
	}

	panic("unreachable")
}

// 100010dw
// only handles register to register
func parseMov(r *Reader) Mov {
	firstByte := r.mustRead()

	result := Mov{}

	switch true {
	// register/memory to/from register
	case (firstByte >> 2) == 0b100010:
		{
			result.typ = Mov_RegisteryOrMemory_ToOrFrom_Register
			result.d = (firstByte >> 1) & 1
			result.w = firstByte & 1
			result.Common = parseCommon(r)
		}
	// immediate to register
	case (firstByte >> 4) == 0b1011:
		{
			result.typ = Mov_Immediate_To_Register
			result.w = (firstByte >> 3) & 1
			result.Common.reg = firstByte & 0b111
			result.data = r.mustReadUint16W(result.w == 1)
		}
	// immediate to register/memory
	case (firstByte >> 1) == 0b1100011:
		{
			result.typ = Mov_Immediate_To_RegisterOrMemory
			result.w = firstByte & 1
			result.Common = parseCommon(r)
			result.data = r.mustReadUint16W(result.w == 1)
		}
	// memory to accumulator
	case (firstByte >> 1) == 0b1010000:
		{
			result.typ = Mov_Memory_To_Accumulator
			result.data = r.mustReadUint16()
		}
	// memory to accumulator
	case (firstByte >> 1) == 0b1010001:
		{
			result.typ = Mov_Accumulator_To_Memory
			result.data = r.mustReadUint16()
		}
	}

	return result
}

func parseArithmetic(r *Reader) Arithmetic {
	firstByte := r.mustRead()

	result := Arithmetic{}
	result.firstByte = firstByte

	switch true {
	case (firstByte >> 2) == 0b100000:
		{
			result.typ = Arithmetic_Immediate_To_RegisterOrMemory
			result.s = (firstByte >> 1) & 1
			result.w = (firstByte >> 0) & 1
			result.Common = parseCommon(r)
			result.data = r.mustReadUint16W(result.s == 0 && result.w == 1)
		}

	case (firstByte>>2)&1 == 0b1:
		{
			result.typ = Arithmetic_Immediate_To_Accumulator
			result.w = (firstByte >> 0) & 1
			result.data = r.mustReadUint16W(result.w == 1)
		}

	case (firstByte>>2)&1 == 0b0:
		{
			result.typ = Arithmetic_RegOrMemory_With_Register_To_Either
			result.d = (firstByte >> 1) & 1
			result.w = (firstByte >> 0) & 1
			result.Common = parseCommon(r)
		}
	}

	return result
}

func parseCommon(r *Reader) Common {
	result := Common{}
	b1 := r.mustRead()

	result.mod = (b1 >> 6) & 0b11
	result.reg = (b1 >> 3) & 0b111
	result.rm = b1 & 0b111

	if result.mod == 0b01 {
		result.disp = int16(int8(r.mustRead()))
	} else if result.mod == 0b10 || (result.mod == 0b00 && result.rm == 0b110) {
		result.disp = r.mustReadInt16()
	}

	return result
}

var ErrOutOfBound = errors.New("out of bound")
