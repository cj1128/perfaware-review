package main

import (
	"fmt"
	"strings"
)

type Flags struct {
	zero bool
	sign bool
}

func (f *Flags) String() string {
	result := &strings.Builder{}

	if f.zero {
		result.WriteString("Z")
	}
	if f.sign {
		result.WriteString("S")
	}

	return result.String()
}

type Sim struct {
	// ax, cx, dx, bx, sp, bp, si, di
	regs  [8]uint16
	flags Flags
}

func (s *Sim) exec(cmd Command) (string, error) {
	switch v := cmd.(type) {
	case *Mov:
		{
			return s.handleMov(cmd.(*Mov))
		}
	case *Arithmetic:
		{
			return s.handleArithmetic(cmd.(*Arithmetic))
		}
	default:
		{
			return "", fmt.Errorf("unsupported type: %v", v)
		}
	}
}

func (s *Sim) handleArithmetic(a *Arithmetic) (string, error) {
	output := &strings.Builder{}
	var result, target, source uint16
	var sourceReg, targetReg byte
	writeBack := true

	switch a.typ {
	case Arithmetic_Immediate_To_RegisterOrMemory:
		{
			assert(a.mod == 0b11, "only support register to register")
			targetReg = a.rm
			target = s.getReg(targetReg, a.w)
			source = a.data
		}
	case Arithmetic_RegOrMemory_With_Register_To_Either:
		{
			assert(a.mod == 0b11, "only support register to register")

			sourceReg = a.reg
			targetReg = a.rm
			if a.d == 1 {
				sourceReg, targetReg = targetReg, sourceReg
			}

			source = s.getReg(sourceReg, a.w)
			target = s.getReg(targetReg, a.w)
		}
	}

	switch a.op {
	case Arithmetic_Cmp:
		writeBack = false
		fallthrough
	case Arithmetic_Sub:
		{
			result = target - source
		}
	case Arithmetic_Add:
		{
			result = target + source
		}
	}

	if writeBack {
		output.WriteString(fmt.Sprintf("%s:0x%x->0x%x", regName(targetReg, a.w), target, result))
		s.setReg(targetReg, result, a.w)
	}

	prevFlags := s.flags.String()

	if result == 0 {
		s.flags.zero = true
	}

	s.flags.sign = result > 0x7fff

	curFlags := s.flags.String()

	if prevFlags != curFlags {
		output.WriteString(" ")
		output.WriteString(fmt.Sprintf("flags:%s->%s", prevFlags, curFlags))
	}

	return output.String(), nil
}

func (s *Sim) handleMov(m *Mov) (string, error) {
	switch m.typ {
	case Mov_Immediate_To_Register:
		{
			regName := m.regName(m.w)
			prev := s.getReg(m.reg, m.w)
			cur := s.setReg(m.reg, m.data, m.w)
			return fmt.Sprintf("%s:0x%x->0x%x", regName, prev, cur), nil
		}
	case Mov_RegisteryOrMemory_ToOrFrom_Register:
		{
			if m.mod != 0b11 {
				break
			}

			sIdx := m.reg
			dIdx := m.rm

			if m.d == 1 {
				t := sIdx
				sIdx = dIdx
				dIdx = t
			}

			regName := regName(dIdx, m.w)
			prev := s.getReg(dIdx, m.w)
			cur := s.setReg(dIdx, s.getReg(sIdx, m.w), m.w)

			return fmt.Sprintf("%s:0x%x->0x%x", regName, prev, cur), nil
		}
	}

	return "", fmt.Errorf("unsupported mov type")
}

// return new value of the whole register
func (s *Sim) setReg(idx byte, val uint16, wide byte) uint16 {
	assert(idx < 8, "getReg: register index must < 8, got %d", idx)

	if wide == 1 {
		s.regs[idx] = val
		return val
	}

	idx = idx % 4
	r := s.regs[idx]

	// high part
	if idx > 3 {
		r = (r & 0x00ff) | (val << 8)
		s.regs[idx] = r
		return r
	}

	// low part
	r = (r & 0xff00) | (val & 0x00ff)
	s.regs[idx] = r
	return r
}
func (s *Sim) getReg(idx byte, wide byte) uint16 {
	assert(idx < 8, "getReg: register index must < 8, got %d", idx)

	if wide == 1 {
		return s.regs[idx]
	}

	r := s.regs[idx%4]

	// high part
	if idx > 3 {
		return r >> 8
	}

	// low part
	return r & 0xff
}

func (s *Sim) dumpRegs() string {
	b := new(strings.Builder)

	b.WriteString("Final registers:\n")

	idxs := []int{0, 3, 1, 2, 4, 5, 6, 7}

	for _, idx := range idxs {
		val := s.regs[idx]
		if val != 0 {
			b.WriteString(fmt.Sprintf("  %s: 0x%04x (%d)\n", REGISTERS_16[idx], val, val))
		}
	}

	return b.String()
}

func (s *Sim) dumpFlags() string {
	b := new(strings.Builder)
	b.WriteString("Flags: ")
	b.WriteString(s.flags.String())
	b.WriteString("\n")
	return b.String()
}
