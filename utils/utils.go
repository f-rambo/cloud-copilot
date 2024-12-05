package utils

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"
)

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func Md5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

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

func IsFileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func IsValidURL(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	return true
}

func GetFileNameByDownloadUrl(url string) string {
	if !IsValidURL(url) {
		return ""
	}
	return url[strings.LastIndex(url, "/")+1:]
}

func DownloadFile(url, filePath string) error {
	if !IsValidURL(url) {
		return fmt.Errorf("url is not http url")
	}
	if !IsFileExist(filePath) {
		err := os.MkdirAll(filePath, os.ModePerm)
		if err != nil {
			return err
		}
	}
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return DownloadFileToWriter(url, file)
}

func DownloadFileToWriter(url string, writer io.Writer) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", response.Status)
	}

	_, err = io.Copy(writer, response.Body)
	if err != nil {
		return err
	}
	return nil
}

func WriteFile(dir, filename, content string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := fmt.Sprintf("%s/%s", dir, filename)

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open or create file: %w", err)
	}
	defer file.Close()

	if _, err := io.WriteString(file, content); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

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

func initRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func getRandomTimeString() string {
	r := initRand()
	randPart := r.Intn(1000)
	timePart := time.Now().Format("20060102150405")
	return fmt.Sprintf("%s%d", timePart, randPart)
}

func ReadLastNLines(file *os.File, n int) (string, int64, error) {
	if n <= 0 {
		return "", 0, fmt.Errorf("invalid number of lines: %d", n)
	}

	stat, err := file.Stat()
	if err != nil {
		return "", 0, err
	}

	fileSize := stat.Size()
	if fileSize == 0 {
		return "", 0, nil
	}

	bufferSize := 1024
	buf := make([]byte, bufferSize)
	lines := make([]string, 0, n)
	offset := int64(0)
	lineCount := 0
	totalLines := int64(0)

	// First, count total lines
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", 0, err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		totalLines++
	}
	if err := scanner.Err(); err != nil {
		return "", 0, err
	}

	// Reset file pointer to the end
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return "", 0, err
	}

	for offset < fileSize {
		readSize := min(bufferSize, int(fileSize-offset))
		offset += int64(readSize)

		_, err := file.Seek(-offset, io.SeekEnd)
		if err != nil {
			return "", 0, err
		}

		_, err = file.Read(buf[:readSize])
		if err != nil {
			return "", 0, err
		}

		// Process last n lines
		for i := readSize - 1; i >= 0; i-- {
			if buf[i] == '\n' || i == 0 {
				if lineCount < n {
					start := i
					if buf[i] == '\n' {
						start++
					}
					line := string(buf[start:readSize])
					if line != "" || i == 0 {
						lines = append([]string{line}, lines...)
						lineCount++
						readSize = i
					}
				} else {
					break
				}
			}
		}
		if lineCount >= n {
			break
		}
	}

	return strings.Join(lines, "\n"), totalLines, nil
}

func InArray(item string, arr []string) bool {
	for _, v := range arr {
		if v == item {
			return true
		}
	}
	return false
}

// ReadFileFromLine reads a file starting from the given line number
// and returns the content read, the last line number read, and any error encountered.
func ReadFileFromLine(filePath string, startLine int64) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentLine int64 = 0
	var content strings.Builder

	// Skip lines until we reach the starting line
	for currentLine < startLine-1 && scanner.Scan() {
		currentLine++
	}

	// Read the rest of the file
	for scanner.Scan() {
		line := scanner.Text()
		content.WriteString(line)
		content.WriteString("\n")
		currentLine++
	}

	if err := scanner.Err(); err != nil {
		return "", 0, err
	}

	// If the file doesn't end with a newline, we still count it as a line
	if content.Len() > 0 && !strings.HasSuffix(content.String(), "\n") {
		currentLine++
	}

	return content.String(), currentLine, nil
}

func GetPortByAddr(addr string) int32 {
	parts := strings.Split(addr, ":")
	if len(parts) == 2 {
		port := parts[1]
		return cast.ToInt32(port)
	}
	return 0
}

func MergePath(paths ...string) string {
	pathArr := make([]string, 0)
	for _, path := range paths {
		pathArr = append(pathArr, strings.Split(path, "/")...)
	}
	return strings.Join(pathArr, "/")
}

func DecodeYaml(yamlContent string, keyVal map[string]string) string {
	for key, val := range keyVal {
		placeholder := "{" + key + "}"
		yamlContent = strings.ReplaceAll(yamlContent, placeholder, val)
	}
	return yamlContent
}
