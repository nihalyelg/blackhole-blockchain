package token

import (
	"testing"
)

func TestToken(t *testing.T) {
	token := NewToken("BlackHole", "BHT", 18, 1000000)

	token.Mint("alice", 500)
	if token.Balances["alice"] != 500 {
		t.Errorf("Expected 500, got %d", token.Balances["alice"])
	}

	err := token.Transfer("alice", "bob", 200)
	if err != nil {
		t.Errorf("Transfer failed: %v", err)
	}

	if token.Balances["bob"] != 200 {
		t.Errorf("Bob should have received 200, got %d", token.Balances["bob"])
	}
}
