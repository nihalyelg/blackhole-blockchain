package wallet

import (
	"time"
)

// MultiSigWallet represents the multisig wallet itself
type MultiSigWallet struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Name         string         `json:"name"`
	Owners       []Owner        `gorm:"foreignKey:WalletID" json:"owners"`
	Threshold    uint           `json:"threshold"`
	Transactions []Transaction  `gorm:"foreignKey:WalletID" json:"transactions"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// Owner represents an individual wallet owner
type Owner struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	WalletID  uint      `json:"wallet_id"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Transaction is a request that needs multisig approval
type Transaction struct {
	ID         uint        `gorm:"primaryKey" json:"id"`
	WalletID   uint        `json:"wallet_id"`
	To         string      `json:"to"`
	Amount     uint64      `json:"amount"`
	Status     string      `json:"status"` // "pending", "executed", "rejected"
	Approvals  []Approval  `gorm:"foreignKey:TransactionID" json:"approvals"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// Approval represents who approved a transaction
type Approval struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TransactionID uint      `json:"transaction_id"`
	OwnerID       uint      `json:"owner_id"`
	CreatedAt     time.Time `json:"created_at"`
}
