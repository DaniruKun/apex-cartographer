package utils

import (
	"errors"
	"path/filepath"
)

func GetMapPath(mapName string) (string, error) {
	if mapName == "olympus" {
		return filepath.Join("resources", "maps", "olympus.png"), nil
	}
	return "", errors.New("unknown map: " + mapName)
}
