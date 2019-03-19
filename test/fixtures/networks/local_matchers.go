package networkfixtures

import (
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

func satisfyAllConfigKeys(chainspec_url bool) types.GomegaMatcher {
	if chainspec_url {
		return SatisfyAll(
			HaveKey("block_explorer_url"),
			HaveKey("chain"),
			HaveKey("chainspec_url"),
			HaveKey("chainspec_abi_url"),
			HaveKey("cloneable_cfg"),

			HaveKey("engine_id"),
			HaveKey("is_ethereum_network"),
			HaveKey("is_load_balanced"),
			HaveKey("json_rpc_url"),
			HaveKey("native_currency"),
			HaveKey("network_id"),
			HaveKey("protocol_id"),
			HaveKey("websocket_url"),
		)
	}
	return SatisfyAll(
		HaveKey("block_explorer_url"),
		HaveKey("chain"),
		HaveKey("chainspec"),
		HaveKey("chainspec_abi"),
		HaveKey("cloneable_cfg"),

		HaveKey("engine_id"),
		HaveKey("is_ethereum_network"),
		HaveKey("is_load_balanced"),
		HaveKey("json_rpc_url"),
		HaveKey("native_currency"),
		HaveKey("network_id"),
		HaveKey("protocol_id"),
		HaveKey("websocket_url"),
	)
}
