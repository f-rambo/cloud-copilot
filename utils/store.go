package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

var ServerNameAsStoreDirName string

const (
	SqlitePackage = "sqlite"
	LogPackage    = "log"
	ConfigPackage = "configs"
)

func InitServerStore() error {
	if ServerNameAsStoreDirName == "" {
		return errors.New("package store dir name is not set")
	}
	ServerNameAsStoreDirName = fmt.Sprintf(".%s", ServerNameAsStoreDirName)
	storePackages := []string{
		SqlitePackage,
		LogPackage,
		ConfigPackage,
	}
	for _, pkg := range storePackages {
		dirPath, err := GetServerStorePathByNames(pkg)
		if err != nil {
			return err
		}
		err = os.MkdirAll(dirPath, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetServerStorePathByNames(packageNames ...string) (string, error) {
	if ServerNameAsStoreDirName == "" {
		return "", errors.New("package store dir name is not set")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if len(packageNames) == 0 {
		return filepath.Join(home, ServerNameAsStoreDirName), nil
	}
	packageNames = append([]string{home, ServerNameAsStoreDirName}, packageNames...)
	return filepath.Join(packageNames...), nil
}

func CreateStoreFile(filePath string) error {
	if filePath != "" && !IsFileExist(filePath) {
		dir, _ := filepath.Split(filePath)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		err = file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
