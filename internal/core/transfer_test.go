package core_test

import (
	"testing"

	"github.com/SkMarcin/Token-Transfer-API/internal/core"
	"github.com/SkMarcin/Token-Transfer-API/internal/database"
	"github.com/stretchr/testify/assert"
)

var (
	InitialSenderAddress = "0x0000000000000000000000000000000000000000"
	RecipientAddress     = "0x1111111111111111111111111111111111111111"
)

func SetupWalletService(t *testing.T) *core.WalletService {
	db := database.SetupTestDB(t)
	database.ClearTables(db)
	if err := database.SeedInitialBalance(db); err != nil {
		t.Fatalf("Failed to re-seed data: %v", err)
	}
	return core.NewWalletService(db)
}

func TestTransferSuccess(t *testing.T) {
	svc := SetupWalletService(t)
	a := assert.New(t)

	const transferAmount = 1000
	newBalance, err := svc.Transfer(InitialSenderAddress, RecipientAddress, transferAmount)

	a.NoError(err)

	expectedBalance := int64(1000000 - transferAmount)
	a.Equal(expectedBalance, newBalance, "Sender's balance should be correctly debited")

	recipient, err := svc.GetWalletByAddress(RecipientAddress)
	a.NoError(err)
	a.NotNil(recipient)
	a.Equal(int64(transferAmount), recipient.Balance, "Recipient's balance should be correctly credited")
}

func TestSequentialTransfers(t *testing.T) {
	svc := SetupWalletService(t)
	a := assert.New(t)

	const firstAmount = 1000
	const secondAmount = 500

	_, err := svc.Transfer(InitialSenderAddress, RecipientAddress, firstAmount)
	a.NoError(err, "First transfer should succeed")

	sender1, _ := svc.GetWalletByAddress(InitialSenderAddress)
	a.Equal(int64(1000000-firstAmount), sender1.Balance, "Sender balance after first transfer is incorrect")

	// Transfer to existing wallet
	newBalance, err := svc.TransferLocking(InitialSenderAddress, RecipientAddress, secondAmount)
	a.NoError(err, "Second transfer should succeed")

	expectedFinalSenderBalance := int64(1000000 - firstAmount - secondAmount)
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

	const transferAmount = 1000001

	// Attempt transfer
	newBalance, err := svc.TransferLocking(InitialSenderAddress, RecipientAddress, transferAmount)

	a.Error(err)
	a.Contains(err.Error(), "insufficient balance", "Should return 'insufficient balance' error")
	a.Equal(int64(0), newBalance, "New balance should be 0 on failure")

	sender, err := svc.GetWalletByAddress(InitialSenderAddress)
	a.NoError(err)
	a.Equal(int64(1000000), sender.Balance, "Sender's balance must be unchanged after failed transfer")
}
