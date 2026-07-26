//go:build windows

package service

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func getPathFromRegistry() (string, error) {
	k, err := registry.OpenKey(registry.CLASSES_ROOT, `osustable.Uri.osu\DefaultIcon`, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("cannot open registry key: %w", err)
	}
	defer func(k registry.Key) {
		err := k.Close()
		if err != nil {
			log.Println("Error closing registry key:", err)
		}
	}(k)

	value, _, err := k.GetStringValue("")
	if err != nil {
		return "", fmt.Errorf("cannot read registry key value: %w", err)
	}

	splitValue := strings.Split(value, ",")
	if len(splitValue) != 2 {
		return "", fmt.Errorf("cannot read registry key value: %w", err)
	}

	first := splitValue[0]
	first = first[1:]
	path := filepath.Dir(first)

	return path, nil
}
