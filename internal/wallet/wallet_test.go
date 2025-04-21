package wallet

import (
	"testing"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/stretchr/testify/assert"
)

func setupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&MultiSigWallet{}, &Owner{}, &Transaction{}, &Approval{}); err != nil {
		return nil, err
	}

	return db, nil
}

func TestCreateWallet(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test DB: %v", err)
	}

	service := &WalletService{DB: db}

	input := CreateWalletInput{
		Name:      "Test Wallet",
		Owners:    []string{"0xOwner1", "0xOwner2", "0xOwner3"},
		Threshold: 2,
	}

	wallet, err := service.CreateWallet(input)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	assert.Equal(t, input.Name, wallet.Name)
	assert.Equal(t, input.Threshold, wallet.Threshold)
	assert.Len(t, wallet.Owners, 3)
	assert.Equal(t, wallet.Owners[0].Address, "0xOwner1")

	// Test: Invalid threshold > owners
	inputInvalidThreshold := CreateWalletInput{
		Name:      "Test Wallet Invalid",
		Owners:    []string{"0xOwner1", "0xOwner2"},
		Threshold: 3,
	}

	_, err = service.CreateWallet(inputInvalidThreshold)
	if err == nil {
		t.Fatal("Expected error when threshold is greater than number of owners, but got nil")
	}
}

func TestSubmitTransaction(t *testing.T) {
    db, err := setupTestDB()
    if err != nil {
        t.Fatalf("Failed to setup test DB: %v", err)
    }

    // Create wallet service
    service := &WalletService{DB: db}

    // Create wallet
    walletInput := CreateWalletInput{
        Name:      "Test Wallet",
        Owners:    []string{"0xOwner1", "0xOwner2", "0xOwner3"},
        Threshold: 2,
    }
    wallet, err := service.CreateWallet(walletInput)
    if err != nil {
        t.Fatalf("Failed to create wallet: %v", err)
    }

    // Submit transaction (this will already create one approval)
    txInput := SubmitTransactionInput{
        WalletID: wallet.ID,
        From:     "0xOwner1",
        To:       "0xRecipient",
        Amount:   100,
    }
    tx, err := service.SubmitTransaction(txInput)
    if err != nil {
        t.Fatalf("Failed to submit transaction: %v", err)
    }

    // Reload the transaction with approvals
    if err := db.Preload("Approvals").First(&tx, tx.ID).Error; err != nil {
        t.Fatalf("Failed to load transaction with approvals: %v", err)
    }

    // Test: Ensure the approval is added correctly
    assert.Len(t, tx.Approvals, 1, "There should be 1 approval")
    assert.Equal(t, wallet.Owners[0].ID, tx.Approvals[0].OwnerID, "The owner should match")

    // Ensure other owners haven't approved
    for _, approval := range tx.Approvals {
        assert.NotEqual(t, wallet.Owners[1].ID, approval.OwnerID, "The second owner should not have approved")
        assert.NotEqual(t, wallet.Owners[2].ID, approval.OwnerID, "The third owner should not have approved")
    }
}

func TestApproveTransaction(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test DB: %v", err)
	}

	// Create wallet service
	service := &WalletService{DB: db}

	// Create wallet with threshold of 3 (important change)
	walletInput := CreateWalletInput{
		Name:      "Test Wallet",
		Owners:    []string{"0xOwner1", "0xOwner2", "0xOwner3"},
		Threshold: 3, // Changed from 2 to 3
	}
	wallet, err := service.CreateWallet(walletInput)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Submit transaction
	txInput := SubmitTransactionInput{
		WalletID: wallet.ID,
		From:     "0xOwner1", 
		To:       "0xRecipient",
		Amount:   100,
	}
	tx, err := service.SubmitTransaction(txInput)
	if err != nil {
		t.Fatalf("Failed to submit transaction: %v", err)
	}

	// Reload the transaction with approvals
	if err := db.Preload("Approvals").First(&tx, tx.ID).Error; err != nil {
		t.Fatalf("Failed to load transaction with approvals: %v", err)
	}

	// Verify that the transaction has one approval (from the submitter)
	assert.Len(t, tx.Approvals, 1, "Transaction should have 1 initial approval")
	assert.Equal(t, "pending", tx.Status, "Transaction should be pending")

	// Now let's have the second owner approve the transaction
	approveInput := ApproveTransactionInput{
		TransactionID: tx.ID,
		OwnerAddress:  "0xOwner2",
	}
	
	updatedTx, err := service.ApproveTransaction(approveInput)
	if err != nil {
		t.Fatalf("Failed to approve transaction: %v", err)
	}

	// Verify transaction now has 2 approvals
	assert.Len(t, updatedTx.Approvals, 2, "Transaction should have 2 approvals")
	
	// Since our threshold is 3, the transaction should still be pending
	assert.Equal(t, "pending", updatedTx.Status, "Transaction should still be pending")

	// Test: Cannot approve a transaction twice
	_, err = service.ApproveTransaction(approveInput)
	assert.Error(t, err, "Should not be able to approve a transaction twice")
	assert.Contains(t, err.Error(), "already approved", "Error should mention transaction is already approved")

	// Test: Non-owner cannot approve
	nonOwnerApproveInput := ApproveTransactionInput{
		TransactionID: tx.ID,
		OwnerAddress:  "0xNonOwner",
	}
	_, err = service.ApproveTransaction(nonOwnerApproveInput)
	assert.Error(t, err, "Non-owner should not be able to approve")
	assert.Contains(t, err.Error(), "only wallet owners", "Error should mention that only owners can approve")

	// Now let's have the third owner approve and check if status changes
	thirdApproveInput := ApproveTransactionInput{
		TransactionID: tx.ID,
		OwnerAddress:  "0xOwner3",
	}
	
	finalTx, err := service.ApproveTransaction(thirdApproveInput)
	if err != nil {
		t.Fatalf("Failed to approve transaction: %v", err)
	}

	// Verify transaction now has 3 approvals
	assert.Len(t, finalTx.Approvals, 3, "Transaction should have 3 approvals")
	assert.Equal(t, "ready_to_execute", finalTx.Status, "Transaction should be ready to execute with 3 approvals")
}

