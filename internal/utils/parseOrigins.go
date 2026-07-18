package utils

import (
	"errors"
	"strings"
)

func ParseOrigins(raw string) ([]string, error) {
	if raw == "" {
		return nil, errors.New("Parameter provided but not set")
	}

	var parsed []string
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimSpace(o); o != "" {
			parsed = append(parsed, o)
		}
	}

	if len(parsed) == 0 {
		return nil, errors.New("Invalid parameter format")
	}

	return parsed, nil
}
