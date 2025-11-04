package core

import (
	"errors"
	"regexp"
)

var (
	addressRegex         = regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	ErrNonPositiveAmount = errors.New("non-positive transfer amount")
	ErrInvalidAddress    = errors.New("invalid address")
)

func ValidateAddress(address string) error {
	if !addressRegex.MatchString(address) {
		return ErrInvalidAddress
	}

	return nil
}

func ValidateTransferAmount(amount int64) error {
	if amount <= 0 {
		return ErrNonPositiveAmount
	}
	return nil
}
