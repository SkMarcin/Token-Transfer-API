package core

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/SkMarcin/Token-Transfer-API/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *WalletService) RaceConditionTransfer(fromAddr string, toAddr string, amount int64) (int64, error) {
	// 0. Validate transfer
	if err := ValidateTransferAmount(amount); err != nil {
		return 0, err
	}
	if err := ValidateAddress(fromAddr); err != nil {
		return 0, fmt.Errorf("invalid sender address: %w", err)
	}
	if err := ValidateAddress(toAddr); err != nil {
		return 0, fmt.Errorf("invalid recipient address: %w", err)
	}
	if fromAddr == toAddr {
		return 0, errors.New("cannot transfer tokens to the same address")
	}

	// 1. Check sender balance
	var sender models.Wallet
	err := s.DB.Where("address = ?", fromAddr).First(&sender).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, fmt.Errorf("sender wallet not found: %s", fromAddr)
		}
		return 0, fmt.Errorf("error retrieving sender: %w", err)
	}

	// 2. Signal balance check done and wait for other threads
	if TestBarrier != nil {
		TestBarrier.Done()
		<-TestRelease
	}

	if sender.Balance < amount {
		return 0, ErrInsufficientBalance
	}

	// 3. Debit sender
	err = s.DB.Model(&models.Wallet{}).Where("address = ?", fromAddr).
		Update("balance", gorm.Expr("balance - ?", amount)).Error

	if err != nil {
		return 0, fmt.Errorf("failed to debit sender: %w", err)
	}

	// 4. Credit recipient
	recipient := models.Wallet{
		Address: toAddr,
		Balance: amount,
	}

	err = s.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "address"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"balance": gorm.Expr("wallets.balance + ?", amount),
		}),
	}).Create(&recipient).Error

	if err != nil {
		return 0, fmt.Errorf("failed to credit recipient: %w", err)
	}

	// 5. Re-fetch new sender balance
	err = s.DB.Where("address = ?", fromAddr).First(&sender).Error
	if err != nil {
		return 0, fmt.Errorf("error retrieving new sender balance: %w", err)
	}

	return sender.Balance, nil
}

func TestTransferRaceConditionFail(t *testing.T) {
	svc := SetupWalletService(t)
	a := assert.New(t)

	transfers := []struct {
		amount int64
	}{
		{amount: 4},
		{amount: 7},
	}

	// Semaphore waiting for goroutines
	var wg sync.WaitGroup

	TestBarrier = &sync.WaitGroup{}
	TestBarrier.Add(len(transfers))
	TestRelease = make(chan struct{})

	var results = make(chan error, len(transfers))

	// Concurrent Goroutines
	for i, tx := range transfers {
		wg.Add(1)

		from := InitialSenderAddress
		to := RecipientAddress
		transferAmt := tx.amount

		go func(index int, f, t string, amount int64) {
			defer wg.Done()
			_, err := svc.RaceConditionTransfer(f, t, amount)
			results <- err
		}(i, from, to, transferAmt)
	}

	// Wait for all threads and release barrier
	TestBarrier.Wait()
	close(TestRelease)

	wg.Wait()
	close(results)

	TestBarrier = nil
	TestRelease = nil

	// Results
	expectedFinalBalance := int64(-1)

	finalSender, _ := svc.GetWalletByAddress(InitialSenderAddress)
	t.Logf("Final balance %d", finalSender.Balance)
	a.True(finalSender.Balance == expectedFinalBalance,
		"Final balance expected to be -1 because of race conditions.")
}
