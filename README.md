# Review for [Performance Aware Programming](https://www.computerenhance.com/p/table-of-contents)

- NASM version 2.16.01 compiled on Dec 23 2022

## Terms

- IPC: Instruction Per Clock
- ILP: Instruction Level Parallelism

## Prologue: The Five Multipliers

### 01 Introduction 02.01

- two ways to increase perofrmance
  - reduce the number of instructions
  - increase the speed of instructions
- stop thinking exclusively about your source language, think about what that it turns to

### 02 Waste 02.02

- Reasons of why program is getting slow
  - **#1**: waste
    - Add two numbers
      - C version: 1 instruction
      - Python version: 181 instruction
        - Much code is run by the interpreter
      - Profiling shows that adding number in Python is 129x slower than C
- Improve the performance of Python code
  - Move the actual work out to bulk ops written in C

### 03 Instructions per Clock 02.05

- "IPC" or "ILP"
  - Instruction Per Clock
  - Instruction Level Parallelism
- "Unrolling" a step, reduce the loop overhead
- Break the serial chaining, give the CPU more things it can do at the same time

### 04 Single Instruction, Multiple Data 02.08

- SSE: Streaming SIMD Extensions
- SSE: 128bit, AVX: 256bit, AVX512: 512bit

### 05 Caching 02.10

- Every `add` is dependent on the `load`
- Typically, L1 and L2 cache are per core, L3 is shared across all cores
- Performance dropped **dramatically**
  - L1: 13.4
  - L2: 7.7
  - L3: 4.4
  - Memory: 1.44

### 06 Multithreading 02.16

- memory bandwidth is shared by all threads, and normally when you have 2 cores, you don't have doubled memory bandwidth

### 07 Python Revisited 02.18

- remember the two ways to increase speed
  - A. reduce the instructions
  - B. increase the speed of which they move into cpu
- five factors
  - A - Reduce Waste
  - B - IPC/ILP
  - A - SIMD
  - B - Caching
  - B - Multithreading
- Python
  - iteration 1
    - `numpy`, basically the same as the naive version
    - `sum`, about 10x speed up
  - iteration 2
    - using [array](https://docs.python.org/3/library/array.html)
    - now `numpy` is quite impressive, about 100x
  - iteration 3
    - Cython, basically this is C

## Interlude

### 01 The Haversine Distance Problem 02.23

- haversine distance: calculate distance of two points on a sphere
- Takes 25 seconds of calculating 10 millions haversine distance
  - 13 seconds to parse input
  - 12 seconds to do the math
- Statistics on my computer
  - Check code at `02.23` dir
  - ```
    Result: 57.97223693479388
    Input = 9.191009044647217 seconds
    Math = 8.063535213470459 seconds
    Total = 17.254544258117676 seconds
    Throughput = 579557.4690589316 haversines/second
    ```

### 02 "Clean" Code, Horrible Performance 02.28

- What is "clean" code?
  - Prefer polymorphism to “if/else” and “switch”
  - Code should not know about the internals of objects it’s working with
  - Functions should be small
  - Functions should do one thing
  - “DRY” - Don’t Repeat Yourself
- just that one change — writing the code the old fashioned way rather than the “clean” code way — gave us an immediate 1.5x performance increase
  - To put that in in hardware terms, it would be like taking an iPhone 14 Pro Max and reducing it to an iPhone 11 Pro Max. It's three or four years of hardware evolution erased because somebody said to use polymorphism instead of switch statements.
- effectively switching from a type-based mindset to a function-based mindset — we get a massive speed increase. We've gone from a switch statement that was merely 1.5x faster to a table-driven version that’s fully 10x faster or more on the exact same problem.
- But for a certain segment of the computing industry, the answer to “why is software so slow” is in large part “because of ‘clean’ code”. The ideas underlying the “clean” code methodology are almost all horrible for performance, and you shouldn’t do them.

## Part 1: 8086/8088

### 01 Instruction Decoding on the 8086 03.02

- In an 8086, a register was literally something that could store 16 bits of data.
- In addition to naming the entire 16-bit register with AX, BX, and so on, you can also refer just to the high 8 or low 8 bits of a register using “L” and “H”.
  - AX: 16 bits
  - AH: high 8 bits
  - AL: low 8 bits
- Read Intel 8086 manual to get more info about instruction structure
- Use Netwide Assembler(NASM) to assemble
- Homework: write a disassembler to disassemble register to register mov instruction
  - Check code at `03.02` dir

### 02 Decoding Multiple Instructions and Suffixes 03.05

- CPU has to look at the first byte to know whethere there is a second byte, and look at the second byte to know whether there is a third byte. "It really is a nasty, dependency-chain process that the CPU has to do to decode these instructions".
  - > To pack instructions into memory as densely as possible, the 8086 and 8088 CPUs utilize an efficient coding technique. Machine instructions vary from one to six bytes in length.
- Load: `mov bx, [75]`, Store: `mov [75], bx`
- The effective address calculation is what we call resolving the address specified by the expression within the brackets.
- `MOD` field, `00`: no displacement, `01`: 1 byte displacement, `02`: two bytes displacement
- For `ax`, `bx`, `cx`, `dx`, we can use their high and low part separtely, but for `sp(stack pointer)`, `bp(base pointer)`, `si(source index)`, `di(destination index)`, we can't.
- Specifal insturtion `Memory to accumulator` and `Accumulator to memory` for using ax.
- Homework: continue to work on `mov` and handle more cases
  - Check code at `03.05` dir
  - Pass listing37 to listing40
  - NOTE
    - `displacement` is a signed 16 bit integer, even though the manual says it's unsigned
    - `accumulator` means `ax`

### 03 Opcode Patterns in 8086 Arithmetic 03.10

- > It turns out add - like almost all the arithmetic operations in 8086 - also has a “mod reg r/m” encoding, so it can get its source from memory just as easily as a register!
- `IP-INC8`: 8-bit signed increment to instruction pointer.
- ADD, Immediate to register/memory
  - `s=0,w=0`: ADD REG8/MEM8, IMMED8
  - `s=0,w=1`: ADD REG16/MEM16, IMMED16
  - `s=1,w=0`: ADD REG8/MEM8, IMMED8
  - `s=1,w=1`: ADD REG16/MEM16, IMMED8
- Homework: decode `add`, `sub` and `cmp`, each one has three different encodings (all of which exactly mirror one of the mov encodings)
  - Check code at `03.10` dir
  - For jumps, we can output `jxx/loopx $+2+[increment]` instead of producing labels
  - Pass listing37 to listing41
  - TODO: listing42

### 04 8086 Decoder Code Review 03.16

- Casey walks through his decoder implementation
  - a table based very flexiable way
  - slow now but it' can be improved

### 05 Using the Reference Decoder as a Shared Library 03.22

- Casey demos how to use his reference decoder as a shared libarry
- I am gonna use my own decoder

### 06 Simulating Non-memory MOVs 03.26

> As I said before, every operation this CPU can do has to be phrased in terms of these registers. Numbers get copied from memory into these registers, get modified by basic operations like addition and multiplication, and then get copied back out.
- Homework: simulate non-memory `mov`
  - Check code at `03.26` dir
  - Pass listing43 to listing44
  - TODO: listing45
