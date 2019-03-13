package networkfixtures

func ethNonProdClonableEnabledEmptyConfigNetwork() (n *NetworkFields) {
	n = &NetworkFields{
		ApplicationID: nil,
		UserID:        nil,
		Name:          ptrTo("Name ETH non-Cloneable Enabled"),
		Description:   ptrTo("Ethereum Network"),
		IsProduction:  ptrToBool(false),
		Cloneable:     ptrToBool(false),
		Enabled:       ptrToBool(true),
		ChainID:       nil,
		SidechainID:   nil,
		NetworkID:     nil,
		Config:        marshalConfig(map[string]interface{}{}),
		Stats:         nil}
	return
}
