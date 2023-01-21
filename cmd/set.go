/*
Copyright © 2022 Zander Hill <zander@xargs.io>
*/
package cmd

import (
	"bufio"
	"fmt"
	"io"
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
	RunE: DoSet,
}

func init() {
	RootCmd.AddCommand(setCmd)
}

func DoSet(cmd *cobra.Command, args []string) error {
	chain := args[0]

	ring, err := NewStore(chain)
	log.Debug().Str("store_type", ring.Name()).Msg("")
	if err != nil {
		return eris.Wrapf(err, "Unable to open keyring for chain: %+v", chain)
	}

	if isInteractive() {
		return processInteractiveEntry(ring)
	} else {
		return processStdinEntry(ring)
	}
}

func isInteractive() bool {
	info, _ := os.Stdin.Stat()
	return (info.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

func processStdinEntry(ring Store) error {
	br := bufio.NewReader(os.Stdin)

	lineCount := 0

	for {
		line, err := br.ReadString('\n')
		if (err != nil) && (err != io.EOF) {
			break
		}

		line = strings.TrimRight(line, "\n")

		log.Debug().Str("line", line).Msg("Line")

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

func processInteractiveEntry(ring Store) error {
	lineCount := 0

	for {
		key, err := promptForString("KEY > ")
		if err != nil {
			return err
		}

		if len(key) == 0 {
			break
		}

		value, err := promptForPassword("VALUE > ")
		if err != nil {
			return err
		}

		lineCount += 1

		err = ring.Set(keyring.Item{
			Key:  key,
			Data: []byte(value),
		})

		if err != nil {
			log.Fatal().Msgf("Unable to set key: %+v +%v", key, err)
		}
	}

	fmt.Printf("Value(s) saved: %d\n", lineCount)
	return nil
}
