package integration_tests

func (suite *PaymentTestSuite) TestKeysendPayment() {
	// destination pubkey strings:
	// simnet-lnd-2: 025c1d5d1b4c983cc6350fc2d756fbb59b4dc365e45e87f8e3afe07e24013e8220
	// simnet-lnd-3: 03c7092d076f799ab18806743634b4c9bb34e351bdebc91d5b35963f3dc63ec5aa
	// simnet-cln-1: 0242898f86064c2fd72de22059c947a83ba23e9d97aedeae7b6dba647123f1d71b
	// (put this in utils)
	// fund account, test making keysend payments to any of these nodes (lnd-2 and lnd-3 is fine)
	// test making a keysend payment to a destination that does not exist
	// test making a keysend payment with a memo that is waaaaaaay too long
}
