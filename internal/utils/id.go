package utils

import (
	"crypto/sha256"
	"fmt"
	"time"
)

func GenerateContainerID() string {
	hash := sha256.Sum256([]byte(time.Now().String()))
	return fmt.Sprintf("%x", hash[:12])
}

func GeneratePodID() string {
	return "pod-" + GenerateContainerID()
}