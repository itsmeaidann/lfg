package utils

import (
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

func LoadEnv(key string) string {
	value, valid := os.LookupEnv(key)

	if !valid {
		log.Fatalf("fail to load env '%v'", key)
	}
	if value == "" {
		log.Fatalf("env '%v' is empty", key)
		return ""
	}

	return value
}

func LoadIntEnv(key string) int {
	value, valid := os.LookupEnv(key)

	if !valid {
		log.Fatalf("fail to load env '%v'", key)
	}
	if value == "" {
		log.Fatalf("env '%v' is empty", key)
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("env '%v' is not integer", key)
	}

	return intValue
}

func LoadBoolEnvWithDefault(key string) bool {
	value, valid := os.LookupEnv(key)
	if !valid || value == "" {
		return false
	}
	return value == "true"
}
