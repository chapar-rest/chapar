package keyvalue

type transactionOnly struct {
	store Store
}

func newFSTransactioner(store Store) *transactionOnly {
	return &transactionOnly{store}
}

func (t *transactionOnly) Transaction(options TransactionOptions) (Transaction, error) {
	return TransactionOrSerial(t.store, options)
}
