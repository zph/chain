/*
Copyright © 2022 Zander Hill <zander@xargs.io>
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/99designs/keyring"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set [keychain]",
	Short: "Set a key in keychain",
	Long: `chain set:
	Set a value in the keychain

	Example:
	$ chain set keychain-name<ENTER>
	$> EXAMPLE_KEY="example-value"
	$> <enter on empty line to save>

	Via pipeline
	$ echo EXAMPLE_KEY="example-value" | chain set keychain-name
	`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		chain := args[0]

		err := set(cmd, chain)
		if err != nil {
			log.Fatal().Msgf(eris.ToString(err, true))
		}
	},
}

func init() {
	RootCmd.AddCommand(setCmd)
}

func set(cmd *cobra.Command, chain string) error {
	ring, err := NewStore(chain)
	log.Debug().Str("store_type", ring.Name()).Msg("")
	if err != nil {
		return eris.Wrapf(err, "Unable to open keyring for chain: %+v", chain)
	}

	br := bufio.NewReader(os.Stdin)

	info, _ := os.Stdin.Stat()

	if (info.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
		// We're in interactive STDIN
		fmt.Print("Enter KEY=value pairs and press enter on empty line to exit:\n")
	}

	lineCount := 0

	// TODO: offer to read in keys and passwords separately to avoid printing them
	// on screen
	for {
		if (info.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
			// We're in interactive STDIN
			fmt.Print(" > ")
		}
		line, err := br.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)

		if len(line) == 0 {
			break
		}

		lineCount += 1

		key, val, found := strings.Cut(line, "=")
		if !found {
			log.Fatal().Msg("Input format must be NAME=VALUE")
		}

		err = ring.Set(keyring.Item{
			Key:  key,
			Data: []byte(val),
		})

		if err != nil {
			log.Fatal().Msgf("Unable to set key: %+v +%v", key, err)
		}
	}

	fmt.Printf("Value(s) saved: %d\n", lineCount)
	return nil
}
