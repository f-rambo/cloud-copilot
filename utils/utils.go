package utils

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

func Md5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func StructTransform(a, b any) error {
	yamlByte, err := yaml.Marshal(a)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(yamlByte, b)
}

// check string '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*'
func IsValidKubernetesName(name string) bool {
	if len(name) == 0 {
		return false
	}

	if !isAlphaNumeric(rune(name[0])) {
		return false
	}

	if !isAlphaNumeric(rune(name[len(name)-1])) {
		return false
	}

	for i := 1; i < len(name)-1; i++ {
		c := rune(name[i])
		if !isAlphaNumeric(c) && c != '-' && c != '.' {
			return false
		}

		if (c == '.' || c == '-') && (name[i+1] == '.' || name[i+1] == '-') {
			return false
		}
	}

	return true
}

func isAlphaNumeric(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

func TransferredMeaning(data any, fileDetailPath string) (tmpFile string, err error) {
	if fileDetailPath == "" {
		return tmpFile, errors.New("fileDetailPath cannot be empty")
	}
	templateByte, err := os.ReadFile(fileDetailPath)
	if err != nil {
		return
	}
	tmpl, err := template.New(filepath.Base(fileDetailPath)).Parse(string(templateByte))
	if err != nil {
		return
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return
	}

	tempFileName := "*-" + filepath.Base(fileDetailPath)
	tempDir := "/tmp"
	tmpFileObj, err := os.CreateTemp(tempDir, tempFileName)
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file")
	}
	defer tmpFileObj.Close()

	if err = os.Chmod(tmpFileObj.Name(), 0666); err != nil {
		return "", errors.Wrap(err, "failed to change file permissions")
	}

	if _, err = tmpFileObj.Write(buf.Bytes()); err != nil {
		return "", errors.Wrap(err, "failed to write to temp file")
	}

	return tmpFileObj.Name(), nil
}

func TransferredMeaningString(data any, fileDetailPath string) (string, error) {
	if fileDetailPath == "" {
		return "", fmt.Errorf("fileDetailPath cannot be empty")
	}
	templateByte, err := os.ReadFile(fileDetailPath)
	if err != nil {
		return "", err
	}
	tmpl, err := template.New(filepath.Base(fileDetailPath)).Parse(string(templateByte))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// example: startIp 192.168.0.1 endIp 192.168.0.254, return 192.168.0.1 192.168.0.2 .... 192.168.0.254
func RangeIps(startIp, endIp string) []string {
	var result []string

	start := ip4ToUint32(startIp)
	end := ip4ToUint32(endIp)

	if start > end {
		return result
	}

	for i := start; i <= end; i++ {
		result = append(result, uint32ToIp4(i))
	}

	return result
}

func ip4ToUint32(ip string) uint32 {
	bits := strings.Split(ip, ".")
	if len(bits) != 4 {
		return 0
	}

	b0, _ := strconv.Atoi(bits[0])
	b1, _ := strconv.Atoi(bits[1])
	b2, _ := strconv.Atoi(bits[2])
	b3, _ := strconv.Atoi(bits[3])

	var sum uint32
	sum += uint32(b0) << 24
	sum += uint32(b1) << 16
	sum += uint32(b2) << 8
	sum += uint32(b3)

	return sum
}

func uint32ToIp4(ipInt uint32) string {
	b0 := ((ipInt >> 24) & 0xFF)
	b1 := ((ipInt >> 16) & 0xFF)
	b2 := ((ipInt >> 8) & 0xFF)
	b3 := (ipInt & 0xFF)

	return fmt.Sprintf("%d.%d.%d.%d", b0, b1, b2, b3)
}

func IsFileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func StringPtr(s string) *string {
	return &s
}

func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func Int64Ptr(i int64) *int64 {
	return &i
}

// ListDirectories
func ListDirectories(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}
	return dirs, nil
}

func DecodeString(templateStr string, envVars map[string]string) string {
	if templateStr == "" {
		return templateStr
	}

	tmpl, err := template.New("env").Parse(templateStr)
	if err != nil {
		return templateStr
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, envVars)
	if err != nil {
		return templateStr
	}

	return buf.String()
}
