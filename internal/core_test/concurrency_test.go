package core_test

import (
	"sync"
	"testing"

	core "github.com/SkMarcin/Token-Transfer-API/internal/core"
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

	core.TestBarrier = &sync.WaitGroup{}
	core.TestBarrier.Add(len(transfers))
	core.TestRelease = make(chan struct{})

	var results = make(chan error, len(transfers))

	// Concurrent Goroutines
	for i, tx := range transfers {
		wg.Add(1)

		from := InitialSenderAddress
		to := RecipientAddress
		transferAmt := tx.amount

		go func(index int, f, t string, amount int64) {
			defer wg.Done()
			_, err := svc.Transfer(f, t, amount)
			results <- err
		}(i, from, to, transferAmt)
	}

	// Wait for all threads and release barrier
	core.TestBarrier.Wait()
	close(core.TestRelease)

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
