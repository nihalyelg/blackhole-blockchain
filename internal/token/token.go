package token

import (
	"errors"
	"sync"
)

// Token represents our blockchain token
type Token struct {
	Name       string
	Symbol     string
	Decimals   uint
	TotalSupply uint64
	Balances   map[string]uint64
	mu         sync.Mutex
}

// NewToken initializes a new token
func NewToken(name, symbol string, decimals uint, initialSupply uint64) *Token {
	return &Token{
		Name:       name,
		Symbol:     symbol,
		Decimals:   decimals,
		TotalSupply: initialSupply,
		Balances:   make(map[string]uint64),
	}
}

// Mint adds new tokens to an address
func (t *Token) Mint(to string, amount uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Balances[to] += amount
	t.TotalSupply += amount
}

// Transfer moves tokens between accounts
func (t *Token) Transfer(from, to string, amount uint64) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Balances[from] < amount {
		return errors.New("insufficient balance")
	}

	t.Balances[from] -= amount
	t.Balances[to] += amount
	return nil
}
