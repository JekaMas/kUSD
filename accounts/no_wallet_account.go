package accounts

// noWalletAccount represents an uninitiated wallet / account or not set
type noWalletAccount struct {
	Wallet
}

func NewNoWalletAccount(wallet Wallet) *noWalletAccount {
	return &noWalletAccount{wallet}
}

func (noAccount *noWalletAccount) Account() Account {
	return Account{}
}
