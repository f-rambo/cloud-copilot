package http

import (
	"github.com/go-resty/resty/v2"
)

func GetContentByUrl(url string) (string, error) {
	client := resty.New()
	resp, err := client.R().Get(url)
	if err != nil {
		return "", err
	}
	return resp.String(), nil
}
