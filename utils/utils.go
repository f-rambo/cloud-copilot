package utils

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

func ArrContains(a, b []string) bool {
	for _, v := range a {
		if slices.Contains(b, v) {
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

func StructTransform(a, b any) error {
	yamlByte, err := yaml.Marshal(a)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(yamlByte, b)
}

func CalculatePercentageInt32(part, total int32) int32 {
	if total == 0 {
		return 0
	}
	partRate := float64(part) / 100
	return int32(math.Ceil(float64(total) * partRate))
}

func RemoveDuplicatesInt64(slice []int64) []int64 {
	seen := make(map[int64]struct{}, len(slice))
	j := 0
	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			slice[j] = v
			j++
		}
	}
	return slice[:j]
}

func DecodeString(content string, keyVal map[string]string) string {
	if content == "" {
		return ""
	}
	for key, val := range keyVal {
		placeholder := "{" + key + "}"
		content = strings.ReplaceAll(content, placeholder, val)
	}
	return content
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

// MapToString converts a map[string]string to a comma-separated string
func MapToString(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}

	pairs := make([]string, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ",")
}

// StringToMap converts a comma-separated string back to map[string]string
func StringToMap(s string) map[string]string {
	if s == "" {
		return make(map[string]string)
	}

	result := make(map[string]string)
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
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

// CreateFile
// Create a file at the specified path. If the file already exists, it will be truncated.
func CreateFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return nil
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

// MergeMaps combines multiple string maps into a single map.
// If there are duplicate keys, the value from the later map takes precedence.
func MergeMaps(maps ...map[string]string) map[string]string {
	if len(maps) == 0 {
		return make(map[string]string)
	}

	// Estimate the capacity by summing the sizes of all input maps
	totalSize := 0
	for _, m := range maps {
		totalSize += len(m)
	}

	result := make(map[string]string, totalSize)
	for _, m := range maps {
		if m == nil {
			continue
		}
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

func Int64Ptr(i int64) *int64 {
	return &i
}

// labels (string) to map[string]string
func LabelsToMap(labels string) map[string]string {
	if labels == "" {
		return make(map[string]string)
	}
	m := make(map[string]string)
	for _, label := range strings.Split(labels, ",") {
		kv := strings.Split(label, "=")
		if len(kv) == 2 {
			m[kv[0]] = kv[1]
		}
	}
	return m
}

// map[string]string to labels (string)
func MapToLabels(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	var labels []string
	for k, v := range m {
		labels = append(labels, k+"="+v)
	}
	return strings.Join(labels, ",")
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
