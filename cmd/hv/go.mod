module github.com/blacktop/go-hypervisor/cmd/hv

go 1.25.0

require (
	github.com/blacktop/go-hypervisor v0.0.0
	github.com/spf13/cobra v1.9.1
	golang.org/x/sys v0.35.0
)

replace github.com/blacktop/go-hypervisor => ../..

require (
	github.com/blacktop/go-dwarf v1.0.14 // indirect
	github.com/blacktop/go-macho v1.1.249 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
)
