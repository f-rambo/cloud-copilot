package utils

import (
	"crypto/md5"
	"encoding/hex"

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
