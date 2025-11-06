package core

import (
	"testing"

	"github.com/SkMarcin/Token-Transfer-API/internal/database"
	"github.com/stretchr/testify/assert"
)

var (
	InitialSenderAddress = "0x0000000000000000000000000000000000000000"
	RecipientAddress     = "0x1111111111111111111111111111111111111111"
)

func SetupWalletService(t *testing.T) *WalletService {
	db := database.SetupTestDB(t)
	database.ClearTables(db)
	if err := database.SeedTestWallets(db); err != nil {
		t.Fatalf("Failed to re-seed data: %v", err)
	}
	return NewWalletService(db)
}

func TestTransferSuccess(t *testing.T) {
	svc := SetupWalletService(t)
	a := assert.New(t)

	const transferAmount = 1
	newBalance, err := svc.Transfer(InitialSenderAddress, RecipientAddress, transferAmount)

	a.NoError(err)

	expectedBalance := int64(10 - transferAmount)
	a.Equal(expectedBalance, newBalance, "Sender's balance should be correctly debited")

	recipient, err := svc.GetWalletByAddress(RecipientAddress)
	a.NoError(err)
	a.NotNil(recipient)
	a.Equal(int64(transferAmount), recipient.Balance, "Recipient's balance should be correctly credited")
}

func TestSequentialTransfers(t *testing.T) {
	svc := SetupWalletService(t)
	a := assert.New(t)

	const firstAmount = 5
	const secondAmount = 3

	_, err := svc.Transfer(InitialSenderAddress, RecipientAddress, firstAmount)
	a.NoError(err, "First transfer should succeed")

	sender1, _ := svc.GetWalletByAddress(InitialSenderAddress)
	a.Equal(int64(10-firstAmount), sender1.Balance, "Sender balance after first transfer is incorrect")

	// Transfer to existing wallet
	newBalance, err := svc.Transfer(InitialSenderAddress, RecipientAddress, secondAmount)
	a.NoError(err, "Second transfer should succeed")

	expectedFinalSenderBalance := int64(10 - firstAmount - secondAmount)
	a.Equal(expectedFinalSenderBalance, newBalance, "Sender's final balance should be correct after both transfers")

	recipient, err := svc.GetWalletByAddress(RecipientAddress)
	a.NoError(err)
	a.NotNil(recipient)
	expectedRecipientBalance := int64(firstAmount + secondAmount)
	a.Equal(expectedRecipientBalance, recipient.Balance, "Recipient's balance should be cumulative")
}

func TestTransferInsufficientBalance(t *testing.T) {
	svc := SetupWalletService(t)
	a := assert.New(t)

	const transferAmount = 11

	// Attempt transfer
	newBalance, err := svc.Transfer(InitialSenderAddress, RecipientAddress, transferAmount)

	a.Error(err)
	a.Contains(err.Error(), "insufficient balance", "Should return 'insufficient balance' error")
	a.Equal(int64(0), newBalance, "New balance should be 0 on failure")

	sender, err := svc.GetWalletByAddress(InitialSenderAddress)
	a.NoError(err)
	a.Equal(int64(10), sender.Balance, "Sender's balance must be unchanged after failed transfer")
}
