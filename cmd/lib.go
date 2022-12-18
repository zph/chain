package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/99designs/keyring"
	"github.com/manifoldco/promptui"
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

// TODO: consider supporting other backends:
// https://pkg.go.dev/github.com/99designs/keyring#BackendType
func getKeyring(chain string) (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		AllowedBackends:  []keyring.BackendType{keyring.FileBackend},
		ServiceName:      chain,
		FilePasswordFunc: getPassword,
		FileDir:          filePath(chain),
	})
}

func set(cmd *cobra.Command, chain string) {
	ring, err := getKeyring(chain)
	if err != nil {
		log.Fatalln("Unable to open keyring")
	}

	br := bufio.NewReader(os.Stdin)

	info, _ := os.Stdin.Stat()

	if (info.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
		// We're in interactive STDIN
		fmt.Print("Enter KEY=value pairs and press enter on empty line to exit:\n")
	}

	lineCount := 0

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
			log.Fatalln("Input format must be NAME=VALUE")
		}

		err = ring.Set(keyring.Item{
			Key:  key,
			Data: []byte(val),
		})

		if err != nil {
			log.Fatalf("Unable to set key: %+v +%v", key, err)
		}
	}

	fmt.Printf("Value(s) saved: %d\n", lineCount)
}

func get(cmd *cobra.Command, chain string) {
	ring, err := getKeyring(chain)

	if err != nil {
		log.Fatalln("Unable to open keyring")
	}

	var lines []string

	var keys []string
	keys, err = ring.Keys()
	if err != nil {
		log.Fatalf("Unable to get keys for keyring\n")
	}
	for _, k := range keys {
		value, err := ring.Get(k)
		if err != nil {
			log.Fatalf("Unable to get key: %+v", k)
		}
		line := fmt.Sprintf("%s=%s", k, value.Data)
		lines = append(lines, line)
	}

	fmt.Println(strings.Join(lines, "\n"))
}

func execute(cmd *cobra.Command, chain string, command string, commandArgs []string) {
	ring, err := getKeyring(chain)

	if err != nil {
		log.Fatalln("Unable to open keyring")
	}

	env := os.Environ()

	var keys []string
	keys, err = ring.Keys()
	if err != nil {
		log.Fatalf("Unable to get keys for keyring\n")
	}
	for _, k := range keys {
		value, err := ring.Get(k)
		if err != nil {
			log.Fatalf("Unable to get key: %+v", k)
		}
		line := fmt.Sprintf("%s=%s", k, value.Data)
		env = append(env, line)
	}

	execpath, err := exec.LookPath(command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to find command: %s\n", command)

		cmd.Help()
		os.Exit(1)
	}

	// TODO: use golang helpers for exec
	err = syscall.Exec(execpath, commandArgs, env)
	log.Fatal(err)
}
