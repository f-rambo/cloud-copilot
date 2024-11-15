package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type File struct {
	path       string
	name       string
	outputFile *os.File
	resume     bool
}

func NewFile(path, name string, resume bool) (*File, error) {
	if !resume {
		name = getRandomTimeString() + filepath.Ext(name)
	}
	f := &File{path: path, name: name, resume: resume}
	err := f.handlerPath()
	if err != nil {
		return nil, err
	}
	err = f.handlerFile()
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (f *File) Write(chunk []byte) error {
	if f.outputFile == nil {
		return fmt.Errorf("file is not open")
	}
	_, err := f.outputFile.Write(chunk)
	return err
}

func (f *File) Read() ([]byte, error) {
	if f.outputFile == nil {
		return nil, fmt.Errorf("file is not open")
	}
	return io.ReadAll(f.outputFile)
}

func (f *File) Close() error {
	if f.outputFile == nil {
		return fmt.Errorf("file is not open")
	}
	err := f.outputFile.Close()
	if err != nil {
		return err
	}
	f.outputFile = nil
	return nil
}

func (f *File) GetFileName() string {
	return f.name
}

func (f *File) GetFilePath() string {
	return f.path
}

func (f *File) GetFileFullPath() string {
	return f.path + f.name
}

func (f *File) ClearFileContent() error {
	return os.Truncate(f.path+f.name, 0)
}

func (f *File) handlerPath() error {
	if f.path == "" {
		return fmt.Errorf("path is empty")
	}
	if f.path[len(f.path)-1:] != "/" {
		f.path += "/"
	}
	if f.checkIsObjExist(f.path) {
		return nil
	}
	return f.createDir()
}

func (f *File) handlerFile() (err error) {
	if f.name == "" {
		return fmt.Errorf("name is empty")
	}
	if f.checkIsObjExist(f.path + f.name) {
		if f.resume {
			f.outputFile, err = os.OpenFile(f.path+f.name, os.O_APPEND|os.O_WRONLY, 0644)
			return err
		}
		err = f.deleteFile()
		if err != nil {
			return err
		}
	}
	f.outputFile, err = f.createFile()
	return err
}

func (f *File) checkIsObjExist(obj string) bool {
	if _, err := os.Stat(obj); os.IsNotExist(err) {
		return false
	}
	return true
}

func (f *File) createDir() error {
	return os.MkdirAll(f.path, os.ModePerm)
}

func (f *File) createFile() (*os.File, error) {
	return os.Create(f.path + f.name)
}

func (f *File) deleteFile() error {
	return os.Remove(f.path + f.name)
}

func GetFilePathAndName(path string) (string, string) {
	fileName := filepath.Base(path)
	filePath := path[:len(path)-len(fileName)]
	return filePath, fileName
}
