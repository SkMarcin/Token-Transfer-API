package core

import (
	"errors"
	"fmt"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	TestBarrier            *sync.WaitGroup
	TestRelease            chan struct{}
)

type WalletService struct {
	DB *gorm.DB
}

func NewWalletService(db *gorm.DB) *WalletService {
	return &WalletService{DB: db}
}

func (s *WalletService) GetWalletByAddress(address string) (*Wallet, error) {
	var wallet Wallet
	result := s.DB.Where("address = ?", address).First(&wallet)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &wallet, nil
}

func (s *WalletService) Transfer(fromAddr string, toAddr string, amount int64) (int64, error) {
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

		// Lock sender row, retrieve balance
		var sender Wallet
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("address = ?", fromAddr).
			First(&sender).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("sender wallet not found: %s", fromAddr)
			}
			return fmt.Errorf("error locking sender wallet: %w", err)
		}

		// Balance check
		if sender.Balance < amount {
			return ErrInsufficientBalance
		}

		// Debit sender
		err = tx.Model(&Wallet{}).Where("address = ?", fromAddr).
			Update("balance", gorm.Expr("balance - ?", amount)).Error
		if err != nil {
			return fmt.Errorf("failed to debit sender: %w", err)
		}

		// Credit recipient
		recipient := Wallet{
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
		var updatedSender Wallet
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
