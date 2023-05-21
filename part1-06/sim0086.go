package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var debugFlag *bool
var execFlag *bool

// check disassemble result by comparing reassemble binary with original binary
var checkFlag *bool

func main() {
	checkFlag = flag.Bool("check", false, "enable check mode")
	debugFlag = flag.Bool("debug", false, "enable debug mode")
	execFlag = flag.Bool("exec", false, "exec")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Println("usage: ./sim0086 [-check] [-debug] [-exec] <binary>")
		os.Exit(0)
	}

	file := flag.Arg(0)
	base := filepath.Base(file)

	cmds, err := disasembleFile(file)

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("; disassembled by sim006: %s\n", file)

	var output []string = []string{"bits 16"}

	sim := new(Sim)

	fmt.Println("bits 16")
	for _, cmd := range cmds {
		var simInfo string

		if *execFlag {
			var err error
			simInfo, err = sim.exec(cmd)
			if err != nil {
				log.Fatalf("failed to do simulation: %v", err)
			}
		}

		str := cmd.Disassemble()
		fmt.Print(str)

		if simInfo != "" {
			fmt.Printf(" ; %s\n", simInfo)
		} else {
			fmt.Print("\n")
		}

		output = append(output, str)
	}

	if *execFlag {
		fmt.Println()
		fmt.Print(sim.dumpRegs())
	}

	// use `nasm` to assemble our disassemble file
	// and then compare it to origin binary
	// source: a
	// our disassemble: a.sim0086.asm
	// nasm reassemble: a.sim0086
	// then compare a and a.sim0086
	if *checkFlag {
		result := strings.Join(output, "\n")
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

// generate binary in the same directory
func nasmAssembleFile(fp string) error {
	command := "nasm"

	cmd := exec.Command(command, fp)

	_, err := cmd.Output()

	if err != nil {
		return err
	}

	return nil
}
