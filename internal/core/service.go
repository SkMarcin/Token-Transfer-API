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
	// 1. Check sender balance
	var sender Wallet
	err := s.DB.Where("address = ?", fromAddr).First(&sender).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, fmt.Errorf("sender wallet not found: %s", fromAddr)
		}
		return 0, fmt.Errorf("error retrieving sender: %w", err)
	}

	// Signal balance check done and wait for other threads
	if TestBarrier != nil {
		TestBarrier.Done()
		<-TestRelease
	}

	if sender.Balance < amount {
		return 0, ErrInsufficientBalance
	}

	// 2. Debit sender
	err = s.DB.Model(&Wallet{}).Where("address = ?", fromAddr).
		Update("balance", gorm.Expr("balance - ?", amount)).Error

	if err != nil {
		return 0, fmt.Errorf("failed to debit sender: %w", err)
	}

	// 3. Credit recipient
	recipient := Wallet{
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

	// 4. Re-fetch new sender balance
	err = s.DB.Where("address = ?", fromAddr).First(&sender).Error
	if err != nil {
		return 0, fmt.Errorf("error retrieving new sender balance: %w", err)
	}

	return sender.Balance, nil
}
