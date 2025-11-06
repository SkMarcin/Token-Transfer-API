package core

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransferRaceCondition(t *testing.T) {
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

	// Concurrent Goroutines
	for i, tx := range transfers {
		wg.Add(1)

		from := InitialSenderAddress
		to := RecipientAddress
		transferAmt := tx.amount

		go func(index int, f, t string, amount int64) {
			defer wg.Done()
			svc.Transfer(f, t, amount)
		}(i, from, to, transferAmt)
	}

	wg.Wait()

	// Results
	expectedFinalBalance1 := int64(6)
	expectedFinalBalance2 := int64(3)

	finalSender, _ := svc.GetWalletByAddress(InitialSenderAddress)
	t.Logf("Final balance %d", finalSender.Balance)
	a.True(finalSender.Balance == expectedFinalBalance1 ||
		finalSender.Balance == expectedFinalBalance2,
		"Final balance must be 6 or 3 after the concurrent transfers.")
}

func TestTransferRaceConditionReceiving(t *testing.T) {
	svc := SetupWalletService(t)

	a := assert.New(t)

	transfers := []struct {
		amount    int64
		receiving bool
	}{
		{amount: 1, receiving: true},
		{amount: 4, receiving: false},
		{amount: 7, receiving: false},
	}

	// Semaphore waiting for goroutines
	var wg sync.WaitGroup

	errorChan := make(chan error, len(transfers))

	// Concurrent Goroutines
	for i, tx := range transfers {
		wg.Add(1)

		from := InitialSenderAddress
		to := RecipientAddress
		transferAmt := tx.amount

		if tx.receiving {
			from = "0x2222222222222222222222222222222222222222"
			to = InitialSenderAddress
		}

		go func(index int, f, t string, amount int64) {
			defer wg.Done()
			_, err := svc.Transfer(f, t, amount)
			errorChan <- err
		}(i, from, to, transferAmt)
	}

	wg.Wait()
	close(errorChan)

	// Error count
	successCount := 0
	insufficientErrorCount := 0

	for err := range errorChan {
		if err == nil {
			successCount++
		} else {
			if errors.Is(err, ErrInsufficientBalance) {
				insufficientErrorCount++
			} else {
				t.Errorf("Received unexpected error: %v", err)
			}
		}
	}

	// Balance count and check
	finalSender, _ := svc.GetWalletByAddress(InitialSenderAddress)
	t.Logf("Final balance %d", finalSender.Balance)
	switch finalSender.Balance {
	case 7, 4:
		a.Equal(2, successCount, "If balance is 4 or 7, exactly two transactions must have succeeded.")
		a.Equal(1, insufficientErrorCount, "If balance is 4 or 7, exactly one transaction must have failed due to insufficient funds.")
	case 0:
		a.Equal(3, successCount, "If balance is 0, all three transactions must have succeeded.")
		a.Equal(0, insufficientErrorCount, "If balance is 0, zero transactions must have failed due to insufficient funds.")
	default:
		a.Failf("Final balance is invalid",
			"Balance was %d. Expected 0, 4, or 7. This indicates an integrity failure.", finalSender.Balance)
	}
}
