package env

import "os"

func IsSet(key string) bool {
	return os.Getenv(key) != ""
}

func IsNotSet(key string) bool {
	return !IsSet(key)
}
