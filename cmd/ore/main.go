package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rsiota/ore/internal/git"
	"github.com/rsiota/ore/internal/ui"
	"github.com/rsiota/ore/internal/version"
)

func main() {
	var (
		versionFlag bool
		pathFlag    string
	)
	flag.BoolVar(&versionFlag, "version", false, "Print version information and exit")
	flag.StringVar(&pathFlag, "C", "", "Run as if started in this directory (git -C style)")
	flag.Parse()

	if versionFlag {
		fmt.Println(version.String())
		fmt.Printf("  go:       %s\n", runtime.Version())
		fmt.Printf("  platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return
	}

	root := pathFlag
	if root == "" {
		if flag.NArg() > 0 {
			root = flag.Arg(0)
		} else {
			var err error
			root, err = os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "ore: %v\n", err)
				os.Exit(1)
			}
		}
	}
	root, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ore: %v\n", err)
		os.Exit(1)
	}

	repo, err := git.Open(root)
	if err != nil {
		if git.IsNotRepository(err) {
			fmt.Fprintf(os.Stderr, "ore: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "ore: %v\n", err)
		os.Exit(1)
	}

	if err := ui.Run(repo); err != nil {
		fmt.Fprintf(os.Stderr, "ore: %v\n", err)
		os.Exit(1)
	}
}
