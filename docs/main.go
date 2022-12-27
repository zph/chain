package main

import (
	"log"

	"github.com/spf13/cobra/doc"
	"github.com/zph/chain/cmd"
)

func main() {
	err := doc.GenMarkdownTree(cmd.RootCmd, "./docs")
	if err != nil {
		log.Fatal(err)
	}
}
