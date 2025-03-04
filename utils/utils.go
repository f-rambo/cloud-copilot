package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"
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

func StructTransform(a, b proto.Message) error {
	protoBuf, err := proto.Marshal(a)
	if err != nil {
		return err
	}
	return proto.Unmarshal(protoBuf, b)
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
