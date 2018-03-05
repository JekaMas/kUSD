package accounts

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewNoWalletAccount(t *testing.T) {
	wallet := &MockWallet{}

	walletAccountInstance := NewNoWalletAccount(wallet)

	assert.IsType(t, &noWalletAccount{}, walletAccountInstance)
}
