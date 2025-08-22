//go:build darwin && arm64

package hypervisor

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"
)

// CPUState matches the structure in cmd/hv/cmd/execute.go
type CPUState struct {
	X0   uint64 `json:"x0"`
	X1   uint64 `json:"x1"`
	X2   uint64 `json:"x2"`
	X3   uint64 `json:"x3"`
	X4   uint64 `json:"x4"`
	X5   uint64 `json:"x5"`
	X6   uint64 `json:"x6"`
	X7   uint64 `json:"x7"`
	X8   uint64 `json:"x8"`
	X9   uint64 `json:"x9"`
	X10  uint64 `json:"x10"`
	X11  uint64 `json:"x11"`
	X12  uint64 `json:"x12"`
	X13  uint64 `json:"x13"`
	X14  uint64 `json:"x14"`
	X15  uint64 `json:"x15"`
	X16  uint64 `json:"x16"`
	X17  uint64 `json:"x17"`
	X18  uint64 `json:"x18"`
	X19  uint64 `json:"x19"`
	X20  uint64 `json:"x20"`
	X21  uint64 `json:"x21"`
	X22  uint64 `json:"x22"`
	X23  uint64 `json:"x23"`
	X24  uint64 `json:"x24"`
	X25  uint64 `json:"x25"`
	X26  uint64 `json:"x26"`
	X27  uint64 `json:"x27"`
	X28  uint64 `json:"x28"`
	FP   uint64 `json:"fp"`
	LR   uint64 `json:"lr"`
	SP   uint64 `json:"sp"`
	PC   uint64 `json:"pc"`
	CPSR uint64 `json:"cpsr"`
}

// ExecuteResult matches the structure in cmd/hv/cmd/execute.go
type ExecuteResult struct {
	State    CPUState          `json:"state"`
	ExitInfo ExitInfo          `json:"exit_info"`
	Memory   map[string][]byte `json:"memory,omitempty"`
	Error    string            `json:"error,omitempty"`
}

// HypervisorTester provides a high-level interface for testing ARM64 code
type HypervisorTester struct {
	hvBinaryPath string
	timeout      time.Duration
}

// NewHypervisorTester creates a new hypervisor tester
func NewHypervisorTester() (*HypervisorTester, error) {
	// Look for hv binary in current directory or PATH
	hvPath := "./hv"
	if _, err := os.Stat(hvPath); os.IsNotExist(err) {
		var err error
		hvPath, err = exec.LookPath("hv")
		if err != nil {
			return nil, err
		}
	}

	return &HypervisorTester{
		hvBinaryPath: hvPath,
		timeout:      5 * time.Second,
	}, nil
}

// ExecuteInstruction executes a single ARM64 instruction and returns the final state
func (ht *HypervisorTester) ExecuteInstruction(initialState *CPUState, instruction []byte) (*CPUState, error) {
	result, err := ht.executeCode(initialState, instruction)
	if err != nil {
		return nil, err
	}
	return &result.State, nil
}

// ExecuteCode executes ARM64 code and returns the complete result
func (ht *HypervisorTester) ExecuteCode(initialState *CPUState, code []byte) (*ExecuteResult, error) {
	return ht.executeCode(initialState, code)
}

func (ht *HypervisorTester) executeCode(initialState *CPUState, code []byte) (*ExecuteResult, error) {
	// Create temporary files for state and code if needed
	var stateFile string
	if initialState != nil {
		tmpFile, err := os.CreateTemp("", "hvtest_state_*.json")
		if err != nil {
			return nil, err
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		stateData, err := json.Marshal(initialState)
		if err != nil {
			return nil, err
		}

		if _, err := tmpFile.Write(stateData); err != nil {
			return nil, err
		}
		stateFile = tmpFile.Name()
	}

	// Prepare command (mem-size defaults to 16384 now)
	args := []string{"execute"}
	if stateFile != "" {
		args = append(args, "--state", stateFile)
	}

	cmd := exec.Command(ht.hvBinaryPath, args...)
	cmd.Stdin = bytes.NewReader(code)

	// Run with timeout
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, err
		}
	case <-time.After(ht.timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return nil, os.ErrDeadlineExceeded
	}

	// Parse result
	var result ExecuteResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, err
	}

	if result.Error != "" {
		return nil, os.ErrInvalid
	}

	return &result, nil
}

// Example test showing how to use the hypervisor tester
func TestHypervisorTester(t *testing.T) {
	// Skip hypervisor tests in CI environments (no nested virtualization support)
	if isCI() {
		t.Skip("Skipping hypervisor tests in CI environment")
	}

	tester, err := NewHypervisorTester()
	if err != nil {
		t.Skip("Hypervisor tester not available (hv binary not found)")
	}

	// Test 1: Simple MOV instruction
	t.Run("MOV X0, #0x42", func(t *testing.T) {
		// mov x0, #0x42; brk #0
		code := []byte{0x40, 0x08, 0x80, 0xd2, 0x00, 0x00, 0x20, 0xd4}

		initialState := &CPUState{
			X0: 100, // Set initial value
			X1: 200,
		}

		finalState, err := tester.ExecuteInstruction(initialState, code)
		if err != nil {
			t.Fatalf("Failed to execute instruction: %v", err)
		}

		// Verify results
		if finalState.X0 != 0x42 {
			t.Errorf("Expected X0=0x42, got X0=0x%x", finalState.X0)
		}
		if finalState.X1 != 200 {
			t.Errorf("Expected X1=200 (unchanged), got X1=%d", finalState.X1)
		}

		t.Logf("Final state: X0=0x%x, X1=%d", finalState.X0, finalState.X1)
	})

	// Test 2: ADD instruction
	t.Run("ADD X0, X1, X2", func(t *testing.T) {
		// add x0, x1, x2; brk #0
		code := []byte{0x20, 0x00, 0x02, 0x8b, 0x00, 0x00, 0x20, 0xd4}

		initialState := &CPUState{
			X1: 10,
			X2: 20,
		}

		finalState, err := tester.ExecuteInstruction(initialState, code)
		if err != nil {
			t.Fatalf("Failed to execute instruction: %v", err)
		}

		expected := uint64(30) // 10 + 20
		if finalState.X0 != expected {
			t.Errorf("Expected X0=%d, got X0=%d", expected, finalState.X0)
		}

		t.Logf("ADD result: %d + %d = %d", initialState.X1, initialState.X2, finalState.X0)
	})

	// Test 3: Complete execution result with memory
	t.Run("Full execution result", func(t *testing.T) {
		code := []byte{0x40, 0x08, 0x80, 0xd2, 0x00, 0x00, 0x20, 0xd4}

		result, err := tester.ExecuteCode(nil, code)
		if err != nil {
			t.Fatalf("Failed to execute code: %v", err)
		}

		// Verify exit info
		if result.ExitInfo.Reason != ExitException {
			t.Errorf("Expected exit reason %d, got %d", ExitException, result.ExitInfo.Reason)
		}

		// Verify memory is captured
		if len(result.Memory) == 0 {
			t.Error("Expected memory to be captured")
		}

		t.Logf("Exit info: Reason=%d, ESR=0x%x", result.ExitInfo.Reason, result.ExitInfo.ESR)
		t.Logf("Memory regions: %d", len(result.Memory))
	})
}

// Example of how to use this in your emulator unit tests
func TestEmulatorVsHypervisor(t *testing.T) {
	if isCI() {
		t.Skip("Skipping hypervisor tests in CI environment")
	}

	tester, err := NewHypervisorTester()
	if err != nil {
		t.Skip("Hypervisor tester not available")
	}

	testCases := []struct {
		name         string
		code         []byte
		initialState *CPUState
		description  string
	}{
		{
			name:         "MOV immediate",
			code:         []byte{0x40, 0x08, 0x80, 0xd2, 0x00, 0x00, 0x20, 0xd4},
			initialState: &CPUState{X0: 999},
			description:  "mov x0, #0x42; brk #0",
		},
		{
			name:         "ADD registers",
			code:         []byte{0x20, 0x00, 0x02, 0x8b, 0x00, 0x00, 0x20, 0xd4},
			initialState: &CPUState{X1: 15, X2: 25},
			description:  "add x0, x1, x2; brk #0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing: %s", tc.description)

			// Get hypervisor result as "source of truth"
			hvResult, err := tester.ExecuteInstruction(tc.initialState, tc.code)
			if err != nil {
				t.Fatalf("Hypervisor execution failed: %v", err)
			}

			// TODO: Add your emulator execution here
			// emulatorResult := yourEmulator.Execute(tc.initialState, tc.code)

			// TODO: Compare states
			// compareStates(t, hvResult, emulatorResult)

			t.Logf("Hypervisor result: X0=0x%x, X1=0x%x, X2=0x%x",
				hvResult.X0, hvResult.X1, hvResult.X2)
		})
	}
}

// Helper to compare CPU states between emulator and hypervisor
func compareStates(t *testing.T, expected, actual *CPUState) {
	if expected.X0 != actual.X0 {
		t.Errorf("X0 mismatch: expected 0x%x, got 0x%x", expected.X0, actual.X0)
	}
	if expected.X1 != actual.X1 {
		t.Errorf("X1 mismatch: expected 0x%x, got 0x%x", expected.X1, actual.X1)
	}
	// Add more register comparisons as needed...
}
