package core_test

import (
	"sync"
	"testing"

	core "github.com/SkMarcin/Token-Transfer-API/internal/core"
	db "github.com/SkMarcin/Token-Transfer-API/internal/database"
	"github.com/stretchr/testify/assert"
)

func TestTransferRaceCondition(t *testing.T) {
	svc := SetupWalletService(t)
	a := assert.New(t)

	transfers := []struct {
		amount int64
	}{
		{amount: 400000},
		{amount: 700000},
	}

	// Semaphore waiting for goroutines
	var wg sync.WaitGroup

	var results = make(chan error, len(transfers))

	// Concurrent Goroutines
	for i, tx := range transfers {
		wg.Add(1)

		from := InitialSenderAddress
		to := RecipientAddress
		transferAmt := tx.amount

		go func(index int, f, t string, amount int64) {
			defer wg.Done()
			_, err := svc.TransferLocking(f, t, amount)
			results <- err
		}(i, from, to, transferAmt)
	}

	wg.Wait()
	close(results)

	core.TestBarrier = nil
	core.TestRelease = nil

	// Results
	expectedFinalBalance1 := int64(600000)
	expectedFinalBalance2 := int64(300000)

	finalSender, _ := svc.GetWalletByAddress(InitialSenderAddress)
	t.Logf("Final balance %d", finalSender.Balance)
	a.True(finalSender.Balance == expectedFinalBalance1 ||
		finalSender.Balance == expectedFinalBalance2,
		"Final balance must be 600000 or 300000 after the concurrent transfers.")
}

func TestTransferRaceConditionReceiving(t *testing.T) {
	svc := SetupWalletService(t)
	db.SeedSecondWalletTest()

	a := assert.New(t)

	transfers := []struct {
		amount    int64
		receiving bool
	}{
		{amount: 100000, receiving: true},
		{amount: 400000, receiving: false},
		{amount: 700000, receiving: false},
	}

	// Semaphore waiting for goroutines
	var wg sync.WaitGroup

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
			svc.TransferLocking(f, t, amount)
		}(i, from, to, transferAmt)
	}

	wg.Wait()

	core.TestBarrier = nil
	core.TestRelease = nil

	// Results
	expectedFinalBalance1 := int64(700000)
	expectedFinalBalance2 := int64(400000)
	expectedFinalBalance3 := int64(0)

	finalSender, _ := svc.GetWalletByAddress(InitialSenderAddress)
	t.Logf("Final balance %d", finalSender.Balance)
	a.True(finalSender.Balance == expectedFinalBalance1 ||
		finalSender.Balance == expectedFinalBalance2 ||
		finalSender.Balance == expectedFinalBalance3,
		"Final balance must be 700000, 400000 or 0 after the concurrent transfers.")
}
