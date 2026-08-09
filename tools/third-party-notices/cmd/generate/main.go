package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	notices "github.com/greaveselliott/mars/tools/third-party-notices"
)

func main() {
	repo := flag.String("repo", "", "MARS repository root")
	write := flag.Bool("write", false, "replace the checked-in notice file")
	flag.Parse()
	if *repo == "" {
		fail("--repo is required")
	}
	root, err := filepath.Abs(*repo)
	if err != nil {
		fail("resolve repository root: %v", err)
	}
	first, err := notices.Generate(root)
	if err != nil {
		fail("generate dependency notices: %v", err)
	}
	second, err := notices.Generate(root)
	if err != nil {
		fail("repeat dependency notice generation: %v", err)
	}
	if !bytes.Equal(first, second) {
		fail("dependency notice generation is not deterministic")
	}
	path := filepath.Join(root, "THIRD_PARTY_NOTICES")
	if *write {
		if err := os.WriteFile(path, first, 0o644); err != nil {
			fail("write THIRD_PARTY_NOTICES: %v", err)
		}
		fmt.Printf("updated THIRD_PARTY_NOTICES sha256=%s\n", notices.Digest(first))
		return
	}
	current, err := os.ReadFile(path)
	if err != nil {
		fail("read THIRD_PARTY_NOTICES: %v", err)
	}
	if !bytes.Equal(current, first) {
		fail("THIRD_PARTY_NOTICES is stale; run the exact dependency notice generation command")
	}
	fmt.Printf("THIRD_PARTY_NOTICES is current sha256=%s\n", notices.Digest(first))
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
