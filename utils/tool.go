package utils

import (
	"archive/tar"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
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
			// resume
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

// 判断切片中是否包含查所要的元素
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// md5加密
func Md5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// yaml to json
func YamlToJson(yamlDatas ...string) (string, error) {
	data := make(map[string]interface{})
	for _, v := range yamlDatas {
		var md map[string]interface{}
		err := yaml.Unmarshal([]byte(v), &md)
		if err != nil {
			return "", err
		}
		for k, v := range md {
			data[k] = v
		}
	}
	jsonStr, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(jsonStr), nil
}

func StructTransform(a, b any) error {
	aJson, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return json.Unmarshal(aJson, b)
}

func TimeParse(timeStr string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", timeStr)
}

func getRandomTimeString() string {
	rand.Seed(time.Now().UnixNano())
	randPart := rand.Intn(1000)                     // Generate a random integer
	timePart := time.Now().Format("20060102150405") // Get the current time in the format YYYYMMDDHHMMSS
	return fmt.Sprintf("%s%d", timePart, randPart)
}

func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// 删除文件
func DeleteFile(path string) error {
	return os.Remove(path)
}

// 修改文件名字
func RenameFile(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// 判断文件是否存在
func IsFileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func IsHttpUrl(url string) bool {
	return url[:4] == "http"
}

// 通过http url下 获取文件名字
func GetFileNameByUrl(url string) string {
	if !IsHttpUrl(url) {
		return ""
	}
	return url[strings.LastIndex(url, "/")+1:]
}

// 通过一个http url下载文件， 文件路径和文件名字
func DownloadFile(url, path, name string) error {
	if !IsHttpUrl(url) {
		return fmt.Errorf("url is not http url")
	}
	if !IsFileExist(path) {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	file, err := os.Create(path + name)
	if err != nil {
		return err
	}
	defer file.Close()
	return DownloadFileToWriter(url, file)
}

// 通过一个http url下载文件， 文件路径和文件名字
func DownloadFileToWriter(url string, writer io.Writer) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	_, err = io.Copy(writer, response.Body)
	if err != nil {
		return err
	}
	return nil
}

// 解压文件 param1: tarball: 压缩文件路径 param2: target: 解压目标路径
func Decompress(tarball, target string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()

	gz, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer gz.Close()

	tarReader := tar.NewReader(gz)

	for {
		header, err := tarReader.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		targetPath := filepath.Join(target, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(targetPath); err != nil {
				if err := os.MkdirAll(targetPath, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer file.Close()
			if _, err := io.Copy(file, tarReader); err != nil {
				return err
			}
		}
	}
}

func GetRandomString() string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	rand.NewSource(time.Now().UnixNano())
	for i := 0; i < 12; i++ {
		result = append(result, bytes[rand.Intn(len(bytes))])
	}
	return string(result)
}

func GetValueFromNestedMap(m map[string]interface{}, key string) (interface{}, bool) {
	keys := strings.Split(key, ".")
	var current interface{} = m

	for _, k := range keys {
		switch typed := current.(type) {
		case map[string]interface{}:
			var ok bool
			if current, ok = typed[k]; !ok {
				return nil, false
			}
		default:
			return nil, false
		}
	}

	return current, true
}
