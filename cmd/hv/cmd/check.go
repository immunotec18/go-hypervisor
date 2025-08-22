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
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/blacktop/go-hypervisor"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check Hypervisor.framework support and entitlement status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ok, err := hypervisor.Supported()
		if err != nil {
			fmt.Printf("hv support: error: %v\n", err)
		} else {
			fmt.Printf("hv support: %v\n", ok)
		}

		exe, _ := os.Executable()
		if exe != "" {
			out, _ := exec.Command("codesign", "-dv", "--entitlements", "-", exe).CombinedOutput()
			entStr := string(out)
			entOK := strings.Contains(entStr, "com.apple.security.hypervisor")
			fmt.Printf("entitlements: hypervisor=%v\n", entOK)
		} else {
			fmt.Println("entitlements: unknown (executable path not found)")
		}

		return nil
	},
}
