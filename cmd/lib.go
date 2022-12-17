package cmd

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

// TODO: setup debug logging
// TODO: use XDG config dir
func filePath(name string) string {
	dir := viper.GetString("dir")
	var path string

	// TODO: make this confirm folder exists
	if dir != "" {
		path = filepath.Join(dir, name)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}

		// TODO pickup here
		path = filepath.Join(home, dir, name)
	}

	os.MkdirAll(filepath.Dir(path), 0755)

	return path
}

var enc = base64.RawURLEncoding

func setupKey() []byte {
	// TODO: rename to be accurate as USERNAME of Keychain holder
	service := viper.GetString(KeyringServiceKey)
	user := viper.GetString(KeyringUserKey)

	skey, err := keyring.Get(service, user)
	if err != nil {
		if err != keyring.ErrNotFound {
			log.Fatal(err)
		}
	} else {
		key, err := enc.DecodeString(skey)
		if err != nil {
			log.Fatal(err)
		}

		return key
	}

	// Ok, make a new key

	key := make([]byte, chacha20poly1305.KeySize)

	_, err = io.ReadFull(rand.Reader, key)
	if err != nil {
		log.Fatal(err)
	}

	err = keyring.Set(service, user, enc.EncodeToString(key))
	if err != nil {
		log.Fatal(err)
	}

	return key
}

// TODO: does not dedupe values, it appends each value
func set(cmd *cobra.Command, chain string) {
	key := setupKey()

	c, err := chacha20poly1305.New(key)
	if err != nil {
		log.Fatal(err)
	}

	path := filePath(chain)

	var data []byte

	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatal(err)
		}
	} else {

		nonce := make([]byte, c.NonceSize())

		_, err = io.ReadFull(f, nonce)
		if err != nil {
			log.Fatal(err)
		}

		ciphertext, err := ioutil.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}

		f.Close()

		data, err = c.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			log.Fatal(err)
		}
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

		idx := strings.IndexByte(line, '=')
		if idx == -1 {
			log.Fatalln("Input format must be NAME=VALUE")
		}

		data = append(data, line...)
		data = append(data, '\n')
	}

	nonce := make([]byte, c.NonceSize())

	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		log.Fatal(err)
	}

	ciphertext := c.Seal(nil, nonce, data, nil)

	err = ioutil.WriteFile(path, append(nonce, ciphertext...), 0600)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Value(s) saved: %d\n", lineCount)
}

// TODO: add get-one command?
func get(cmd *cobra.Command, chain string, command string, commandArgs []string) {
	key := setupKey()

	c, err := chacha20poly1305.New(key)
	if err != nil {
		log.Fatal(err)
	}

	path := filePath(chain)

	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	env := os.Environ()

	nonce := make([]byte, c.NonceSize())

	_, err = io.ReadFull(f, nonce)
	if err != nil {
		log.Fatal(err)
	}

	ciphertext, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	data, err := c.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		log.Fatalln("Unable to decrypt chain, perhaps key was changed")
	}

	br := bufio.NewReader(bytes.NewReader(data))

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)

		idx := strings.IndexByte(line, '=')
		if idx != -1 {
			env = append(env, line)
		}
	}

	f.Close()

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

func export(cmd *cobra.Command, args []string) {
	key := setupKey()
	fmt.Println(enc.EncodeToString(key))
}
