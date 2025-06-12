package utils

import (
	"bytes"
	"context"

	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/pkg/errors"
)

type File struct {
	path       string
	name       string
	outputFile *os.File
	resume     bool
}

func initRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func getRandomTimeString() string {
	r := initRand()
	randPart := r.Intn(1000)
	timePart := time.Now().Format("20060102150405")
	return fmt.Sprintf("%s%d", timePart, randPart)
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

func GetServerStoragePathByNames(packageNames ...string) string {
	if len(packageNames) == 0 {
		return ""
	}
	return filepath.Join(packageNames...)
}

func AcceptingFile(ctx context.Context, fileName, uploadDir string) (string, error) {
	httpReq, ok := kratosHttp.RequestFromServerContext(ctx)
	if !ok {
		return "", errors.New("failed to get http request")
	}

	// Parse multipart form
	if err := httpReq.ParseMultipartForm(64 << 20); err != nil {
		return "", errors.Wrap(err, "failed to parse multipart form")
	}

	// Get uploaded file
	file, header, err := httpReq.FormFile(fileName)
	if err != nil {
		return "", errors.Wrap(err, "failed to get uploaded file")
	}
	defer file.Close()

	if uploadDir != "" {
		if err = os.MkdirAll(uploadDir, 0755); err != nil {
			return "", errors.Wrap(err, "failed to create upload directory")
		}
	}

	// Create destination file
	dst, err := os.Create(filepath.Join(uploadDir, header.Filename))
	if err != nil {
		return "", errors.Wrap(err, "failed to create destination file")
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, file); err != nil {
		return "", errors.Wrap(err, "failed to copy file content")
	}

	return header.Filename, nil
}

func AcceptingFileByte(ctx context.Context, fileName string) ([]byte, string, error) {
	httpReq, ok := kratosHttp.RequestFromServerContext(ctx)
	if !ok {
		return nil, "", errors.New("failed to get http request")
	}

	// Parse multipart form
	if err := httpReq.ParseMultipartForm(64 << 20); err != nil {
		return nil, "", errors.Wrap(err, "failed to parse multipart form")
	}

	// Get uploaded file
	file, header, err := httpReq.FormFile(fileName)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to get uploaded file")
	}
	defer file.Close()

	// Copy file content
	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to read file content")
	}
	return buf, header.Filename, nil
}

func UploadFile(serverUrl, localFilepath string) error {

	file, err := os.Open(localFilepath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(localFilepath))
	if err != nil {
		return fmt.Errorf("failed to create form field: %v", err)
	}

	if _, err = io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file content: %v", err)
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %v", err)
	}

	req, err := http.NewRequest("POST", serverUrl, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload failed, status code: %d, response: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
