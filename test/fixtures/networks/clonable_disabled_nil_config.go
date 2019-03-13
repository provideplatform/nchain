package networkfixtures

func ethNonProdClonableDisabledNilConfigNetwork() (n *NetworkFields) {
	n = &NetworkFields{
		ApplicationID: nil,
		UserID:        nil,
		Name:          ptrTo("Name ETH non-Cloneable Disabled"),
		Description:   ptrTo("Ethereum Network"),
		IsProduction:  ptrToBool(false),
		Cloneable:     ptrToBool(false),
		Enabled:       ptrToBool(false),
		ChainID:       nil,
		SidechainID:   nil,
		NetworkID:     nil,
		Config:        nil,
		Stats:         nil}
	return
}
