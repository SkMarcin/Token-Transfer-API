package core

import (
	"errors"
	"fmt"
	"sort"

	"github.com/SkMarcin/Token-Transfer-API/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
)

type WalletService struct {
	DB *gorm.DB
}

func NewWalletService(db *gorm.DB) *WalletService {
	return &WalletService{DB: db}
}

func (s *WalletService) GetWalletByAddress(address string) (*models.Wallet, error) {
	var wallet models.Wallet
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

	// Sort addresses
	var finalSenderBalance int64
	addresses := []string{fromAddr, toAddr}
	sort.Strings(addresses)

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
