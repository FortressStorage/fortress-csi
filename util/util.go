package util

import (
	"crypto/sha512"
	"encoding/hex"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	SizeAlignment     = 2 * 1024 * 1024
	MinimalVolumeSize = 10 * 1024 * 1024
)

// AutoCorrectName converts name to lowercase, and correct overlength name by
// replaces the name suffix with 8 char from its checksum to ensure uniquenedoss.
func AutoCorrectName(name string, maxLength int) string {
	newName := strings.ToLower(name)
	if len(name) > maxLength {
		logrus.Warnf("Name %v is too long, auto-correct to fit %v characters", name, maxLength)
		checksum := GetStringChecksum(name)
		newNameSuffix := "-" + checksum[:8]
		newNamePrefix := strings.TrimRight(newName[:maxLength-len(newNameSuffix)], "-")
		newName = newNamePrefix + newNameSuffix
	}
	if newName != name {
		logrus.Warnf("Name auto-corrected from %v to %v", name, newName)
	}
	return newName
}

func GetStringChecksum(data string) string {
	return GetChecksumSHA512([]byte(data))
}

func GetChecksumSHA512(data []byte) string {
	checksum := sha512.Sum512(data)
	return hex.EncodeToString(checksum[:])
}

func RoundUpSize(size int64) int64 {
	if size <= 0 {
		return SizeAlignment
	}
	r := size % SizeAlignment
	if r == 0 {
		return size
	}
	return size - r + SizeAlignment
}
