package test

import (
	"errors"
)

func GetTestError() error {
	return errors.New("test error")
}

func GetValidUUID() string {
	return "12345678-0000-0000-0000-000000000000"
}
