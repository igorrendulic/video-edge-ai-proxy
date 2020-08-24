package utils

import (
	"errors"
	"net/url"
	"strings"
)

// ParseRTMPKey takes the complete RTMP url and extracts the streaming key
func ParseRTMPKey(rtmpURL string) (string, error) {
	u, err := url.Parse(rtmpURL)
	if err != nil {
		return "", err
	}
	if u.Scheme != "rtmp" {
		return "", err
	}
	splitted := strings.Split(u.Path, "/")
	if len(splitted) > 0 {
		key := splitted[len(splitted)-1]
		return key, nil

	}
	return "", errors.New("failed to parse RTMP key")
}
