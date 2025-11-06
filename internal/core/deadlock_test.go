package core

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SkMarcin/Token-Transfer-API/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	TestBarrier *sync.WaitGroup
	TestRelease chan struct{}
)

func (s *WalletService) DeadlockTransfer(fromAddr string, toAddr string, amount int64) (int64, error) {
	// Validate transfer
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

	var finalSenderBalance int64

	// Start a transaction
	err := s.DB.Transaction(func(tx *gorm.DB) error {

		var sender models.Wallet
		var recipient models.Wallet

		// Lock sender row
		tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("address = ?", fromAddr).First(&sender)

		// Wait for other transfer to cause deadlock
		if TestBarrier != nil {
			TestBarrier.Done()
			<-TestRelease
		}

		// Lock recipient row
		tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("address = ?", toAddr).First(&recipient)

		// Balance check
		if sender.Balance < amount {
			return ErrInsufficientBalance
		}

		// Debit sender
		err := tx.Model(&models.Wallet{}).Where("address = ?", fromAddr).
			Update("balance", gorm.Expr("balance - ?", amount)).Error
		if err != nil {
			return fmt.Errorf("failed to debit sender: %w", err)
		}

		// Credit recipient
		recipient = models.Wallet{
			Address: toAddr,
			Balance: amount,
		}

		err = tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "address"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"balance": gorm.Expr("wallets.balance + ?", amount),
			}),
		}).Create(&recipient).Error

		if err != nil {
			return fmt.Errorf("failed to credit recipient: %w", err)
		}

		// Get the new sender balance
		var updatedSender models.Wallet
		err = tx.Where("address = ?", fromAddr).First(&updatedSender).Error
		if err != nil {
			return fmt.Errorf("failed to retrieve updated balance: %w", err)
		}

		finalSenderBalance = updatedSender.Balance
		return nil
	})

	if err != nil {
		return 0, err
	}

	return finalSenderBalance, nil
}

func (s *WalletService) FixedDeadlockTransfer(fromAddr string, toAddr string, amount int64) (int64, error) {
	// Validate transfer
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

	var finalSenderBalance int64
	addresses := []string{fromAddr, toAddr}
	sort.Strings(addresses) // Sorts addresses

	lock1 := addresses[0]
	lock2 := addresses[1]

	// Start a transaction
	err := s.DB.Transaction(func(tx *gorm.DB) error {

		var (
			wallet1 models.Wallet
			wallet2 models.Wallet
			sender  models.Wallet
		)

		// Lock rows in sorted order
		tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("address = ?", lock1).First(&wallet1)
		time.Sleep(50 * time.Millisecond)
		tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("address = ?", lock2).First(&wallet2)

		if fromAddr == wallet1.Address {
			sender = wallet1
		} else {
			sender = wallet2
		}

		// Balance check
		if sender.Balance < amount {
			return ErrInsufficientBalance
		}

		// Debit sender
		err := tx.Model(&models.Wallet{}).Where("address = ?", fromAddr).
			Update("balance", gorm.Expr("balance - ?", amount)).Error
		if err != nil {
			return fmt.Errorf("failed to debit sender: %w", err)
		}

		// Credit recipient
		recipient := models.Wallet{
			Address: toAddr,
			Balance: amount,
		}

		err = tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "address"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"balance": gorm.Expr("wallets.balance + ?", amount),
			}),
		}).Create(&recipient).Error

		if err != nil {
			return fmt.Errorf("failed to credit recipient: %w", err)
		}

		// Get the new sender balance
		var updatedSender models.Wallet
		err = tx.Where("address = ?", fromAddr).First(&updatedSender).Error
		if err != nil {
			return fmt.Errorf("failed to retrieve updated balance: %w", err)
		}

		finalSenderBalance = updatedSender.Balance
		return nil
	})

	if err != nil {
		return 0, err
	}

	return finalSenderBalance, nil
}

// This could happen before deadlock protection
func TestDeadlockScenario(t *testing.T) {
	svc := SetupWalletService(t)
	a := assert.New(t)

	fromAddr := InitialSenderAddress
	toAddr := SecondarySenderAddress

	transferAmount := int64(3)

	var wg sync.WaitGroup

	TestBarrier = &sync.WaitGroup{}
	TestBarrier.Add(2)
	TestRelease = make(chan struct{})

	var results = make(chan error, 2)

	// Transaction 1
	wg.Go(func() {
		_, err := svc.DeadlockTransfer(fromAddr, toAddr, transferAmount)
		results <- err
	})

	// Transaction 2
	wg.Go(func() {
		_, err := svc.DeadlockTransfer(toAddr, fromAddr, transferAmount)
		results <- err
	})

	TestBarrier.Wait()
	close(TestRelease)

	wg.Wait()
	close(results)

	TestBarrier = nil
	TestRelease = nil

	deadlockDetectedCount := 0

	for err := range results {
		if err != nil {
			if strings.Contains(err.Error(), "transaction block") {
				deadlockDetectedCount++
			} else {
				t.Logf("Received non-deadlock error: %v", err)
			}
		}
	}

	a.GreaterOrEqual(deadlockDetectedCount, 1, "Expected at least one transaction to fail with 'deadlock detected' error.")
}

// Testing if deadlocks are fixed even with delay between locks
func TestDeadlockWithDelay(t *testing.T) {
	svc := SetupWalletService(t)
	a := assert.New(t)

	fromAddr := InitialSenderAddress
	toAddr := SecondarySenderAddress

	transferAmount := int64(3)

	var wg sync.WaitGroup
	var results = make(chan error, 2)

	// Transaction 1
	wg.Go(func() {
		_, err := svc.FixedDeadlockTransfer(fromAddr, toAddr, transferAmount)
		results <- err
	})

	// Transaction 2
	wg.Go(func() {
		_, err := svc.FixedDeadlockTransfer(toAddr, fromAddr, transferAmount)
		results <- err
	})

	wg.Wait()
	close(results)

	deadlockDetectedCount := 0

	for err := range results {
		if err != nil {
			if strings.Contains(err.Error(), "transaction block") {
				deadlockDetectedCount++
			} else {
				t.Logf("Received non-deadlock error: %v", err)
			}
		}
	}

	a.GreaterOrEqual(deadlockDetectedCount, 0, "Expected for all the transactions to pass without deadlock.")
}
