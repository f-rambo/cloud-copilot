package utils

import (
	"crypto/md5"
	"encoding/hex"
	"math"
	"strings"

	"google.golang.org/protobuf/proto"
)

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func ArrContains(a, b []string) bool {
	for _, v := range a {
		if Contains(b, v) {
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
