package wallet

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type WalletService struct {
	DB *gorm.DB
}

type CreateWalletInput struct {
	Name      string
	Owners    []string
	Threshold uint
}

type SubmitTransactionInput struct {
	WalletID uint
	From     string
	To       string
	Amount   uint64
}

type ApproveTransactionInput struct {
	TransactionID uint
	OwnerAddress  string
}

func (s *WalletService) CreateWallet(input CreateWalletInput) (*MultiSigWallet, error) {
	if len(input.Owners) < int(input.Threshold) {
		return nil, errors.New("threshold cannot be greater than number of owners")
	}

	wallet := &MultiSigWallet{
		Name:      input.Name,
		Threshold: input.Threshold,
	}

	if err := s.DB.Create(wallet).Error; err != nil {
		return nil, err
	}

	for _, address := range input.Owners {
		owner := Owner{
			WalletID: wallet.ID,
			Address:  address,
		}
		if err := s.DB.Create(&owner).Error; err != nil {
			return nil, err
		}
	}

	if err := s.DB.Preload("Owners").First(wallet, wallet.ID).Error; err != nil {
		return nil, err
	}

	return wallet, nil
}

func (s *WalletService) SubmitTransaction(input SubmitTransactionInput) (*Transaction, error) {
	var wallet MultiSigWallet
	if err := s.DB.Preload("Owners").First(&wallet, input.WalletID).Error; err != nil {
		return nil, fmt.Errorf("wallet not found: %w", err)
	}

	isOwner := false
	var proposerID uint
	for _, owner := range wallet.Owners {
		if owner.Address == input.From {
			isOwner = true
			proposerID = owner.ID
			break
		}
	}

	if !isOwner {
		return nil, fmt.Errorf("only wallet owners can submit transactions")
	}

	tx := Transaction{
		WalletID: wallet.ID,
		To:       input.To,
		Amount:   input.Amount,
		Status:   "pending",
	}

	if err := s.DB.Create(&tx).Error; err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	approval := Approval{
		TransactionID: tx.ID,
		OwnerID:       proposerID,
	}

	if err := s.DB.Create(&approval).Error; err != nil {
		return nil, fmt.Errorf("failed to auto-approve transaction: %w", err)
	}

	return &tx, nil
}

func (s *WalletService) ApproveTransaction(input ApproveTransactionInput) (*Transaction, error) {
	// Fetch the transaction with its related wallet info
	var transaction Transaction
	if err := s.DB.Preload("Approvals").First(&transaction, input.TransactionID).Error; err != nil {
		return nil, fmt.Errorf("transaction not found: %w", err)
	}

	// Make sure transaction is still pending
	if transaction.Status != "pending" {
		return nil, fmt.Errorf("only pending transactions can be approved")
	}

	// Get wallet with its owners
	var wallet MultiSigWallet
	if err := s.DB.Preload("Owners").First(&wallet, transaction.WalletID).Error; err != nil {
		return nil, fmt.Errorf("wallet not found: %w", err)
	}

	// Verify that the approver is one of the wallet owners
	var ownerID uint
	isOwner := false
	for _, owner := range wallet.Owners {
		if owner.Address == input.OwnerAddress {
			isOwner = true
			ownerID = owner.ID
			break
		}
	}

	if !isOwner {
		return nil, fmt.Errorf("only wallet owners can approve transactions")
	}

	// Check if this owner has already approved this transaction
	for _, approval := range transaction.Approvals {
		if approval.OwnerID == ownerID {
			return nil, fmt.Errorf("transaction already approved by this owner")
		}
	}

	// Create the approval
	approval := Approval{
		TransactionID: transaction.ID,
		OwnerID:       ownerID,
	}

	if err := s.DB.Create(&approval).Error; err != nil {
		return nil, fmt.Errorf("failed to create approval: %w", err)
	}

	// Check if we've reached the approval threshold
	if err := s.DB.Preload("Approvals").First(&transaction, transaction.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload transaction: %w", err)
	}

	// If we've reached the threshold, update the transaction status
	if uint(len(transaction.Approvals)) >= wallet.Threshold {
		transaction.Status = "ready_to_execute" // or you could automatically execute it here
		if err := s.DB.Save(&transaction).Error; err != nil {
			return nil, fmt.Errorf("failed to update transaction status: %w", err)
		}
	}

	return &transaction, nil
}