package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/manifoldco/promptui"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TODO: setup debug logging
// TODO: use XDG config dir
func filePath(name string) string {
	dir := viper.GetString(ChainDirKey)
	var path string

	if dir != "" {
		path = filepath.Join(dir, name)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}

		path = filepath.Join(home, dir, name)
	}

	return path
}

func getPassword(_s string) (string, error) {
	validate := func(input string) error {
		if len(input) < viper.GetInt(PasswordValidationLength) {
			return errors.New("password must have more than 20 characters")
		}
		return nil
	}
	p := viper.GetString(KeyringPassword)
	if p == "" {
		prompt := promptui.Prompt{
			Label:    "Password",
			Validate: validate,
			Mask:     '*',
		}

		result, err := prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return "", err
		}

		return result, nil
	} else {
		if err := validate(p); err != nil {
			return "", err
		} else {
			return p, nil
		}
	}

}

func getKVAsEnvLines(cmd *cobra.Command, chain string) ([]string, error) {
	ring, err := NewStore(chain)

	if err != nil {
		return nil, eris.Wrap(err, "Unable to open keyring")
	}

	var lines []string

	var keys []string
	keys, err = ring.Keys()
	if err != nil {
		return nil, eris.Wrap(err, "Unable to get keys for keyring\n")
	}
	for _, k := range keys {
		value, err := ring.Get(k)
		if err != nil {
			return nil, eris.Wrapf(err, "Unable to get key: %+v", k)
		}
		line := fmt.Sprintf("%s=%s", k, value.Data)
		lines = append(lines, line)
	}

	return lines, err
}
