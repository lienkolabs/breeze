package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"syscall"

	"github.com/lienkolabs/breeze/vault"
	"golang.org/x/term"
)

type Wallet struct {
	vault *vault.SecureVault
	data  os.File
}

func NewConfig() {
	var addres string
	fmt.Printf("instruction gateway address: ")
	fmt.Scan(&addres)
	fmt.Printf("secret password: ")
	_, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

}

func abort(err error) {
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
}

func GetDotSocialPath() string {
	path, err := os.UserHomeDir()
	abort(err)
	path = filepath.Join(path, ".social")
	if _, err := os.ReadDir(path); err != nil {
		err = os.Mkdir(path, fs.ModePerm)
		abort(err)
	}
	return path
}

func Help() {
	fmt.Println(helpdoc)
}

func OpenVault(path string) *vault.SecureVault {
	fmt.Printf("secret password: ")
	passwd, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println("")
	abort(err)
	files, err := os.ReadDir(path)
	abort(err)
	path = filepath.Join(path, "vault")
	for _, file := range files {
		if file.Name() == "vault" {
			if existing := vault.OpenVaultFromPassword(passwd, path); existing != nil {
				return existing
			}
			log.Fatal("could not open secure vault")
		}
	}
	newVault := vault.NewSecureVault(string(passwd), path)
	if newVault == nil {
		log.Fatal("could not create secure vault")
	}
	return newVault
}

func main() {
	if len(os.Args) == 1 {
		Help()
		return
	}
	path := GetDotSocialPath()
	wallet := Wallet{
		vault: OpenVault(path),
	}
	if wallet.vault == nil {
		fmt.Print("opa")
	}
	if len(os.Args) == 1 {
		Help()
	}
}
