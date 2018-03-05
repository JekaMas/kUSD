package accounts

import (
	"github.com/kowala-tech/kUSD/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewWalletAccountWithoutAccountReturnsNoWalletAccount(t *testing.T) {
	address := common.Address{}
	account := Account{Address: address}
	wallet := &MockWallet{}

	walletAccount, err := NewWalletAccount(wallet, account)
	assert.NoError(t, err)

	assert.IsType(t, &noWalletAccount{}, walletAccount)
}

func TestNewWalletAccount(t *testing.T) {
	address := common.Address{1}
	account := Account{Address: address}
	wallet := &MockWallet{}
	wallet.On("Contains", account).Return(true)

	walletAccountInstance, err := NewWalletAccount(wallet, account)
	assert.NoError(t, err)

	assert.IsType(t, &walletAccount{}, walletAccountInstance)
	assert.Equal(t, account, walletAccountInstance.Account())
}

func TestNewWalletAccountFailsIfAddressDoesntExistInWallet(t *testing.T) {
	address := common.Address{1}
	account := Account{Address: address}
	wallet := &MockWallet{}
	wallet.On("Contains", account).Return(false)

	_, err := NewWalletAccount(wallet, account)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidAccountAddress{account}, err)
}
