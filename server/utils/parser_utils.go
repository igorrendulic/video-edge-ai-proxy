package utils

import (
	"errors"
	"net/url"
	"strings"

	"github.com/chryscloud/video-edge-ai-proxy/models"
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

func StringPairsToVarPairs(stringPairs []string) []*models.VarPair {
	varPairs := make([]*models.VarPair, 0)
	if len(stringPairs) > 0 {
		for _, str := range stringPairs {
			if strings.Contains(str, "=") {
				pair := &models.VarPair{}
				split := strings.Split(str, "=")
				name := split[0]
				val := split[1]
				pair.Name = name
				pair.Value = val
				varPairs = append(varPairs, pair)
			}
		}
	}
	return varPairs
}

// ImageTag to part splits the complete image tag to username, registry and version
func ImageTagToParts(imageTag string) (string, string, string) {
	// best effort return
	if !strings.Contains(imageTag, "/") {
		return imageTag, "", ""
	}
	splitUserRegistry := strings.Split(imageTag, "/")
	if len(splitUserRegistry) == 2 {
		dockerUser := splitUserRegistry[0]
		part2 := splitUserRegistry[1]
		registryVersion := strings.Split(part2, ":")
		if len(registryVersion) == 2 {
			dockerRepository := registryVersion[0]
			dockerVersion := registryVersion[1]
			return dockerUser, dockerRepository, dockerVersion
		} else {
			return imageTag, "", ""
		}
	}

	return imageTag, "", ""
}

// combine image parts to complete image tag string
func ImageTagPartToString(dockerUser, dockerRepository, dockerImageVersion string) string {
	return dockerUser + "/" + dockerRepository + ":" + dockerImageVersion
}
