/*
Copyright © 2025 blacktop

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/blacktop/go-hypervisor"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

// CPUState represents the CPU register and memory state
type CPUState struct {
	// General-purpose registers
	X0  uint64 `json:"x0"`
	X1  uint64 `json:"x1"`
	X2  uint64 `json:"x2"`
	X3  uint64 `json:"x3"`
	X4  uint64 `json:"x4"`
	X5  uint64 `json:"x5"`
	X6  uint64 `json:"x6"`
	X7  uint64 `json:"x7"`
	X8  uint64 `json:"x8"`
	X9  uint64 `json:"x9"`
	X10 uint64 `json:"x10"`
	X11 uint64 `json:"x11"`
	X12 uint64 `json:"x12"`
	X13 uint64 `json:"x13"`
	X14 uint64 `json:"x14"`
	X15 uint64 `json:"x15"`
	X16 uint64 `json:"x16"`
	X17 uint64 `json:"x17"`
	X18 uint64 `json:"x18"`
	X19 uint64 `json:"x19"`
	X20 uint64 `json:"x20"`
	X21 uint64 `json:"x21"`
	X22 uint64 `json:"x22"`
	X23 uint64 `json:"x23"`
	X24 uint64 `json:"x24"`
	X25 uint64 `json:"x25"`
	X26 uint64 `json:"x26"`
	X27 uint64 `json:"x27"`
	X28 uint64 `json:"x28"`

	// Special registers
	FP   uint64 `json:"fp"`   // Frame pointer (x29)
	LR   uint64 `json:"lr"`   // Link register (x30)
	SP   uint64 `json:"sp"`   // Stack pointer
	PC   uint64 `json:"pc"`   // Program counter
	CPSR uint64 `json:"cpsr"` // Current program status register
}

// ExecuteResult represents the execution result
type ExecuteResult struct {
	State    CPUState            `json:"state"`
	ExitInfo hypervisor.ExitInfo `json:"exit_info"`
	Memory   map[string][]byte   `json:"memory,omitempty"` // hex address -> data
	Error    string              `json:"error,omitempty"`
}

var (
	stateFile string
	memSize   int
	baseAddr  uint64
)

func init() {
	rootCmd.AddCommand(executeCmd)
	executeCmd.Flags().StringVarP(&stateFile, "state", "s", "", "JSON file with initial CPU state")
	executeCmd.Flags().IntVar(&memSize, "mem-size", 16384, "Memory size to allocate (bytes)")
	executeCmd.Flags().Uint64VarP(&baseAddr, "base-addr", "a", 0x4000, "Base address for code execution")
}

var executeCmd = &cobra.Command{
	Use:   "execute [code-file]",
	Short: "Execute ARM64 code and return CPU state as JSON",
	Long: `Execute ARM64 machine code and return the resulting CPU state as JSON.
	
Code can be provided as:
  - A binary file argument
  - Stdin (if no file argument provided)

Initial CPU state can be provided via --state flag pointing to a JSON file.
Results are output as JSON to stdout.`,
	RunE: runExecute,
}

func runExecute(cmd *cobra.Command, args []string) error {
	// Check hypervisor support
	ok, err := hypervisor.Supported()
	if err != nil || !ok {
		return fmt.Errorf("hypervisor not supported: %v", err)
	}

	// Read initial state if provided
	var initialState CPUState
	if stateFile != "" {
		stateData, err := os.ReadFile(stateFile)
		if err != nil {
			return fmt.Errorf("failed to read state file: %w", err)
		}
		if err := json.Unmarshal(stateData, &initialState); err != nil {
			return fmt.Errorf("failed to parse state JSON: %w", err)
		}
	}

	// Read code input
	var codeData []byte
	if len(args) > 0 {
		// Read from file
		codeData, err = os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read code file: %w", err)
		}
	} else {
		// Read from stdin
		codeData, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
	}

	if len(codeData) == 0 {
		return fmt.Errorf("no code provided")
	}

	// Execute the code
	result, err := executeCode(codeData, &initialState)
	if err != nil {
		result = &ExecuteResult{Error: err.Error()}
	}

	// Output JSON result
	output, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

func executeCode(code []byte, initialState *CPUState) (*ExecuteResult, error) {
	// Create VM
	vm, err := hypervisor.NewVM()
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}
	defer vm.Close()

	// Create vCPU
	vcpu, err := vm.NewVCPU()
	if err != nil {
		return nil, fmt.Errorf("failed to create vCPU: %w", err)
	}
	defer vcpu.Close()

	// Validate memory size is page-aligned
	page := unix.Getpagesize()
	if memSize%page != 0 {
		return nil, fmt.Errorf("mem-size must be a multiple of page size (%d bytes)", page)
	}

	// Allocate memory
	hostMem, err := unix.Mmap(-1, 0, memSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory: %w", err)
	}
	defer unix.Munmap(hostMem)

	// Copy code to memory
	if len(code) > len(hostMem) {
		return nil, fmt.Errorf("code size (%d) exceeds memory size (%d)", len(code), len(hostMem))
	}
	copy(hostMem, code)

	// Map memory into guest
	perms := hypervisor.MemRead | hypervisor.MemWrite | hypervisor.MemExec
	err = vm.Map(hostMem, baseAddr, perms)
	if err != nil {
		return nil, fmt.Errorf("failed to map memory: %w", err)
	}
	defer vm.Unmap(baseAddr, uint64(len(hostMem)))

	// Set initial CPU state
	if err := setCPUState(vcpu, initialState); err != nil {
		return nil, fmt.Errorf("failed to set initial state: %w", err)
	}

	// Set PC to base address if not set in initial state
	if initialState.PC == 0 {
		if err := vcpu.SetPC(baseAddr); err != nil {
			return nil, fmt.Errorf("failed to set PC: %w", err)
		}
	}

	// Execute
	exitInfo, err := vcpu.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to execute: %w", err)
	}

	// Get final CPU state
	finalState, err := getCPUState(vcpu)
	if err != nil {
		return nil, fmt.Errorf("failed to get final state: %w", err)
	}

	// Copy the executed memory to avoid marshaling mmap'd memory
	memCopy := make([]byte, len(code))
	copy(memCopy, hostMem[:len(code)])

	return &ExecuteResult{
		State:    *finalState,
		ExitInfo: exitInfo,
		Memory:   map[string][]byte{fmt.Sprintf("0x%x", baseAddr): memCopy},
	}, nil
}

// setCPUState sets the CPU registers from the state struct
func setCPUState(vcpu *hypervisor.VCPU, state *CPUState) error {
	regs := map[hypervisor.Reg]uint64{
		hypervisor.RegX0:   state.X0,
		hypervisor.RegX1:   state.X1,
		hypervisor.RegX2:   state.X2,
		hypervisor.RegX3:   state.X3,
		hypervisor.RegX4:   state.X4,
		hypervisor.RegX5:   state.X5,
		hypervisor.RegX6:   state.X6,
		hypervisor.RegX7:   state.X7,
		hypervisor.RegX8:   state.X8,
		hypervisor.RegX9:   state.X9,
		hypervisor.RegX10:  state.X10,
		hypervisor.RegX11:  state.X11,
		hypervisor.RegX12:  state.X12,
		hypervisor.RegX13:  state.X13,
		hypervisor.RegX14:  state.X14,
		hypervisor.RegX15:  state.X15,
		hypervisor.RegX16:  state.X16,
		hypervisor.RegX17:  state.X17,
		hypervisor.RegX18:  state.X18,
		hypervisor.RegX19:  state.X19,
		hypervisor.RegX20:  state.X20,
		hypervisor.RegX21:  state.X21,
		hypervisor.RegX22:  state.X22,
		hypervisor.RegX23:  state.X23,
		hypervisor.RegX24:  state.X24,
		hypervisor.RegX25:  state.X25,
		hypervisor.RegX26:  state.X26,
		hypervisor.RegX27:  state.X27,
		hypervisor.RegX28:  state.X28,
		hypervisor.RegFP:   state.FP,
		hypervisor.RegLR:   state.LR,
		hypervisor.RegSP:   state.SP,
		hypervisor.RegPC:   state.PC,
		hypervisor.RegCPSR: state.CPSR,
	}

	for reg, val := range regs {
		if val != 0 { // Only set non-zero values
			if err := vcpu.SetReg(reg, val); err != nil {
				return fmt.Errorf("failed to set %v: %w", reg, err)
			}
		}
	}

	return nil
}

// getCPUState retrieves all CPU registers into a state struct
func getCPUState(vcpu *hypervisor.VCPU) (*CPUState, error) {
	state := &CPUState{}

	regs := []hypervisor.Reg{
		hypervisor.RegX0, hypervisor.RegX1, hypervisor.RegX2, hypervisor.RegX3,
		hypervisor.RegX4, hypervisor.RegX5, hypervisor.RegX6, hypervisor.RegX7,
		hypervisor.RegX8, hypervisor.RegX9, hypervisor.RegX10, hypervisor.RegX11,
		hypervisor.RegX12, hypervisor.RegX13, hypervisor.RegX14, hypervisor.RegX15,
		hypervisor.RegX16, hypervisor.RegX17, hypervisor.RegX18, hypervisor.RegX19,
		hypervisor.RegX20, hypervisor.RegX21, hypervisor.RegX22, hypervisor.RegX23,
		hypervisor.RegX24, hypervisor.RegX25, hypervisor.RegX26, hypervisor.RegX27,
		hypervisor.RegX28, hypervisor.RegFP, hypervisor.RegLR, hypervisor.RegSP,
		hypervisor.RegPC, hypervisor.RegCPSR,
	}

	for _, reg := range regs {
		val, err := vcpu.GetReg(reg)
		if err != nil {
			return nil, fmt.Errorf("failed to get %v: %w", reg, err)
		}

		switch reg {
		case hypervisor.RegX0:
			state.X0 = val
		case hypervisor.RegX1:
			state.X1 = val
		case hypervisor.RegX2:
			state.X2 = val
		case hypervisor.RegX3:
			state.X3 = val
		case hypervisor.RegX4:
			state.X4 = val
		case hypervisor.RegX5:
			state.X5 = val
		case hypervisor.RegX6:
			state.X6 = val
		case hypervisor.RegX7:
			state.X7 = val
		case hypervisor.RegX8:
			state.X8 = val
		case hypervisor.RegX9:
			state.X9 = val
		case hypervisor.RegX10:
			state.X10 = val
		case hypervisor.RegX11:
			state.X11 = val
		case hypervisor.RegX12:
			state.X12 = val
		case hypervisor.RegX13:
			state.X13 = val
		case hypervisor.RegX14:
			state.X14 = val
		case hypervisor.RegX15:
			state.X15 = val
		case hypervisor.RegX16:
			state.X16 = val
		case hypervisor.RegX17:
			state.X17 = val
		case hypervisor.RegX18:
			state.X18 = val
		case hypervisor.RegX19:
			state.X19 = val
		case hypervisor.RegX20:
			state.X20 = val
		case hypervisor.RegX21:
			state.X21 = val
		case hypervisor.RegX22:
			state.X22 = val
		case hypervisor.RegX23:
			state.X23 = val
		case hypervisor.RegX24:
			state.X24 = val
		case hypervisor.RegX25:
			state.X25 = val
		case hypervisor.RegX26:
			state.X26 = val
		case hypervisor.RegX27:
			state.X27 = val
		case hypervisor.RegX28:
			state.X28 = val
		case hypervisor.RegFP:
			state.FP = val
		case hypervisor.RegLR:
			state.LR = val
		case hypervisor.RegSP:
			state.SP = val
		case hypervisor.RegPC:
			state.PC = val
		case hypervisor.RegCPSR:
			state.CPSR = val
		}
	}

	return state, nil
}
