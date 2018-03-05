package accounts

// WalletAccount wraps wallet with a "default" account
type WalletAccount interface {
	Wallet
	Account() Account
}

type walletAccount struct {
	Wallet
	account Account
}

// NewWalletAccount ensure that the address provided exists in the wallet
func NewWalletAccount(wallet Wallet, account Account) (WalletAccount, error) {
	emptyAccount := Account{}
	if account == emptyAccount {
		return NewNoWalletAccount(wallet), nil
	}

	if !wallet.Contains(account) {
		return nil, NewErrInvalidAccountAddress(account)
	}
	return &walletAccount{Wallet: wallet, account: account}, nil
}

func (account *walletAccount) Account() Account {
	return account.account
}
