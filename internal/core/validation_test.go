package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAddress(t *testing.T) {
	a := assert.New(t)

	// Correct
	validAddr := "0x0000000000000000000000000000000000000000"
	a.NoError(ValidateAddress(validAddr), "Valid address should pass validation")

	validAddrMixed := "0xAbCdEf1234567890aBcDeF1234567890aBcDeF12"
	a.NoError(ValidateAddress(validAddrMixed), "Mixed-case hex address should pass validation")

	// Wrong Length
	shortAddr := "0x0000"
	err := ValidateAddress(shortAddr)
	a.ErrorIs(err, ErrInvalidAddress, "Too short address should fail")

	longAddr := "0x00000000000000000000000000000000000000000"
	err = ValidateAddress(longAddr)
	a.ErrorIs(err, ErrInvalidAddress, "Too long address should fail")

	// Wrong Format
	noPrefix := "000000000000000000000000000000000000000000"
	err = ValidateAddress(noPrefix)
	a.ErrorIs(err, ErrInvalidAddress, "Address without 0x prefix should fail")
}

func TestValidateTransferAmount(t *testing.T) {
	a := assert.New(t)
	a.NoError(ValidateTransferAmount(1), "Small positive amount should pass")

	err := ValidateTransferAmount(0)
	a.ErrorIs(err, ErrNonPositiveAmount, "Zero amount should fail")

	err = ValidateTransferAmount(-100)
	a.ErrorIs(err, ErrNonPositiveAmount, "Negative amount should fail")
}
