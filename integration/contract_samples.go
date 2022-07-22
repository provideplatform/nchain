//go:build integration || nchain || ropsten || rinkeby || kovan || goerli || nobookie || readonly
// +build integration nchain ropsten rinkeby kovan goerli nobookie readonly

/*
 * Copyright 2017-2022 Provide Technologies Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package integration

const ekhoArtifact = `{
	"contractName": "Ekho",
	"abi": [
		{
			"anonymous": false,
			"inputs": [
				{
					"indexed": false,
					"internalType": "bytes",
					"name": "message",
					"type": "bytes"
				}
			],
			"name": "Ekho",
			"type": "event"
		},
		{
			"inputs": [
				{
					"internalType": "bytes",
					"name": "message",
					"type": "bytes"
				}
			],
			"name": "broadcast",
			"outputs": [],
			"stateMutability": "nonpayable",
			"type": "function"
		}
	],
	"bytecode": "0x608060405234801561001057600080fd5b506101c0806100206000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c80630323a8b014610030575b600080fd5b6100e96004803603602081101561004657600080fd5b810190808035906020019064010000000081111561006357600080fd5b82018360208201111561007557600080fd5b8035906020019184600183028401116401000000008311171561009757600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f8201169050808301925050505050505091929192905050506100eb565b005b7fc11b5aba5095eee01588a1b4f1983f055885ce9581af8e25b9fb34220d24eb73816040518080602001828103825283818151815260200191508051906020019080838360005b8381101561014d578082015181840152602081019050610132565b50505050905090810190601f16801561017a5780820380516001836020036101000a031916815260200191505b509250505060405180910390a15056fea26469706673582212209a157717b400c42ec8447983d186638a51c9ee770e53cbfce55399bf1d381f8164736f6c63430006090033",
	"source": "contract Ekho {\n    event Ekho(bytes message);\n    function broadcast(bytes memory message) public {\n        emit Ekho(message);\n    }\n}\n"
}`

const ShuttleArtifact = `{
	"contractName":"Shuttle",
	"abi": [
		{
			"inputs": [
				{
					"internalType": "string",
					"name": "name",
					"type": "string"
				},
				{
					"internalType": "address",
					"name": "verifier",
					"type": "address"
				},
				{
					"internalType": "uint256",
					"name": "treeHeight",
					"type": "uint256"
				}
			],
			"stateMutability": "nonpayable",
			"type": "constructor"
		},
		{
			"anonymous": false,
			"inputs": [
				{
					"indexed": false,
					"internalType": "string",
					"name": "subject",
					"type": "string"
				},
				{
					"indexed": false,
					"internalType": "string",
					"name": "name",
					"type": "string"
				},
				{
					"indexed": false,
					"internalType": "address",
					"name": "verifier",
					"type": "address"
				},
				{
					"indexed": false,
					"internalType": "address",
					"name": "shield",
					"type": "address"
				}
			],
			"name": "ShuttleDeployedCircuit",
			"type": "event"
		},
		{
			"anonymous": false,
			"inputs": [
				{
					"indexed": false,
					"internalType": "string",
					"name": "subject",
					"type": "string"
				},
				{
					"indexed": false,
					"internalType": "string",
					"name": "contractType",
					"type": "string"
				},
				{
					"indexed": false,
					"internalType": "string",
					"name": "name",
					"type": "string"
				},
				{
					"indexed": false,
					"internalType": "address",
					"name": "addr",
					"type": "address"
				}
			],
			"name": "ShuttleDeployedContract",
			"type": "event"
		}
	],
	"bytecode": "0x6080604052600060281b600460006101000a8154817affffffffffffffffffffffffffffffffffffffffffffffffffffff021916908360281c021790555034801561004957600080fd5b506040516109453803806109458339818101604052604081101561006c57600080fd5b810190808051906020019092919080519060200190929190505050808060008190555060005460020a6001819055505081602660006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505050610857806100ee6000396000f3fe608060405234801561001057600080fd5b50600436106100885760003560e01c80636e0c3fee1161005b5780636e0c3fee1461020357806376c601b114610255578063bc1b392d14610273578063d7b0fef1146102a157610088565b806301e3e9151461008d57806330e69fc3146100ab57806346657fe9146100c95780636620e73b14610113575b600080fd5b6100956102bf565b6040518082815260200191505060405180910390f35b6100b36102c5565b6040518082815260200191505060405180910390f35b6100d16102cb565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6101e96004803603606081101561012957600080fd5b810190808035906020019064010000000081111561014657600080fd5b82018360208201111561015857600080fd5b8035906020019184602083028401116401000000008311171561017a57600080fd5b90919293919293908035906020019064010000000081111561019b57600080fd5b8201836020820111156101ad57600080fd5b803590602001918460208302840111640100000000831117156101cf57600080fd5b9091929391929390803590602001909291905050506102f5565b604051808215151515815260200191505060405180910390f35b61022f6004803603602081101561021957600080fd5b810190808035906020019092919050505061047e565b604051808264ffffffffff191664ffffffffff1916815260200191505060405180910390f35b61025d61049e565b6040518082815260200191505060405180910390f35b61027b6104a4565b604051808264ffffffffff191664ffffffffff1916815260200191505060405180910390f35b6102a96104b7565b6040518082815260200191505060405180910390f35b60005481565b60035481565b6000602660009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b600080602660009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663b864f5a9888888886040518563ffffffff1660e01b81526004018080602001806020018381038352878782818152602001925060200280828437600081840152601f19601f8201169050808301925050508381038252858582818152602001925060200280828437600081840152601f19601f8201169050808301925050509650505050505050602060405180830381600087803b1580156103d357600080fd5b505af11580156103e7573d6000803e3d6000fd5b505050506040513d60208110156103fd57600080fd5b8101908080519060200190929190505050905080610466576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260368152602001806107c96036913960400191505060405180910390fd5b61046f836104bd565b50600191505095945050505050565b6005816021811061048b57fe5b016000915054906101000a900460281b81565b60015481565b600460009054906101000a900460281b81565b60025481565b60006003546001541161051b576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260238152602001806107ff6023913960400191505060405180910390fd5b600061052860035461072f565b9050600060018054600354010390506000602885901b905060008061054b6107a6565b600080600090505b6000548110156106a857878114156105a757856005896021811061057357fe5b0160006101000a8154817affffffffffffffffffffffffffffffffffffffffffffffffffffff021916908360281c02179055505b6000600288816105b357fe5b06141561063457600581602181106105c757fe5b0160009054906101000a900460281b945085935060405185815284601b8201526020846036836002600019fa9250826000811461060357610605565bfe5b505060288360006001811061061657fe5b6020020151901b95506002600188038161062c57fe5b04965061069b565b859450600460009054906101000a900460281b935060405185815284601b8201526020846036836002600019fa9250826000811461067157610673565bfe5b505060288360006001811061068457fe5b6020020151901b95506002878161069757fe5b0496505b8080600101915050610553565b50816000600181106106b657fe5b60200201516002819055507f6a82ba2aa1d2c039c41e6e2b5a5a1090d09906f060d32af9c1ac0beff7af75c06003548a60025460405180848152602001838152602001828152602001935050505060405180910390a1600360008154809291906001019190505550600254975050505050505050919050565b600080905060016002838161074057fe5b0614156107a1576000600190506000600290506000600182901b90505b600084141561079d576000818360018801038161077657fe5b06141561078557829350610798565b809150600181901b905082806001019350505b61075d565b5050505b919050565b604051806020016040528060019060208202803683378082019150509050509056fe5468652070726f6f66206661696c656420766572696669636174696f6e20696e2074686520766572696669657220636f6e74726163745468657265206973206e6f207370616365206c65667420696e2074686520747265652ea26469706673582212205b68cc9ccdc7b7ef2e92fabde0dfd489efa619e88eaf2f700bf2f3f0731aff7964736f6c63430006090033",
	"source": "// SPDX-License-Identifier: CC0\npragma solidity ^0.6.9;\n\nimport \"./lib/MerkleTreeSHA256.sol\";\nimport \"./IShield.sol\";\nimport \"./IVerifier.sol\";\n\ncontract Shield is IShield, MerkleTreeSHA256 {\n    // CONTRACT INSTANCES:\n    IVerifier private verifier; // the verification smart contract\n\n    // FUNCTIONS:\n    constructor(address _verifier, uint _treeHeight) public MerkleTreeSHA256(_treeHeight) {\n        verifier = IVerifier(_verifier);\n    }\n\n    // returns the verifier contract address that this shield contract uses for proof verification\n    function getVerifier() external override view returns (address) {\n        return address(verifier);\n    }\n\n    function verifyAndPush(\n        uint256[] calldata _proof,\n        uint256[] calldata _publicInputs,\n        bytes32 _newCommitment\n    ) external override returns (bool) {\n\n        // verify the proof\n        bool result = verifier.verify(_proof, _publicInputs);\n        require(result, \"The proof failed verification in the verifier contract\");\n\n        // update contract states\n        insertLeaf(_newCommitment); // recalculate the root of the merkleTree as it's now different\n        return true;\n    }\n\n}\n"
}`

// const ERC165Artifact = `{
// 	"contractName":"ERC165",
// 	"abi": [

// 	],
// 	"bytecode": ,
// 	"source":"contract ERC165 is IERC165 {\n    /*\n     * bytes4(keccak256('supportsInterface(bytes4)')) == 0x01ffc9a7\n     */\n    bytes4 private constant _INTERFACE_ID_ERC165 = 0x01ffc9a7;\n\n    /**\n     * @dev Mapping of interface ids to whether or not it's supported.\n     */\n    mapping(bytes4 => bool) private _supportedInterfaces;\n\n    constructor () internal {\n        // Derived contracts need only register support for their own interfaces,\n        // we register support for ERC165 itself here\n        _registerInterface(_INTERFACE_ID_ERC165);\n    }\n\n    /**\n     * @dev See {IERC165-supportsInterface}.\n     *\n     * Time complexity O(1), guaranteed to always use less than 30 000 gas.\n     */\n    function supportsInterface(bytes4 interfaceId) external view returns (bool) {\n        return _supportedInterfaces[interfaceId];\n    }\n\n    /**\n     * @dev Registers the contract as an implementer of the interface defined by\n     * `interfaceId`. Support of the actual ERC165 interface is automatic and\n     * registering its interface id is not required.\n     *\n     * See {IERC165-supportsInterface}.\n     *\n     * Requirements:\n     *\n     * - `interfaceId` cannot be the ERC165 invalid interface (`0xffffffff`).\n     */\n    function _registerInterface(bytes4 interfaceId) internal {\n        require(interfaceId != 0xffffffff, \"ERC165: invalid interface id\");\n        _supportedInterfaces[interfaceId] = true;\n    }\n}"
// }`

const RegistryArtifact = `{
	"contractName":"Registry",
	"abi": [
		{
			"constant": false,
			"inputs": [
				{
					"name": "_subject",
					"type": "string"
				},
				{
					"name": "_hash",
					"type": "bytes"
				}
			],
			"name": "publish",
			"outputs": null,
			"payable": false,
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": [
				{
					"name": "_page",
					"type": "uint256"
				},
				{
					"name": "_rpp",
					"type": "uint256"
				}
			],
			"name": "listMessages",
			"outputs": [
				{
					"components": [
						{
							"name": "sender",
							"type": "address"
						},
						{
							"name": "timestamp",
							"type": "uint256"
						},
						{
							"name": "subject",
							"type": "string"
						},
						{
							"name": "hash",
							"type": "bytes"
						}
					],
					"name": "_msgs",
					"type": "tuple[]"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"anonymous": false,
			"inputs": [
				{
					"indexed": false,
					"name": "subject",
					"type": "string"
				},
				{
					"indexed": false,
					"name": "key",
					"type": "bytes32"
				},
				{
					"indexed": false,
					"name": "sender",
					"type": "address"
				}
			],
			"name": "Published",
			"type": "event"
		}
	],
	"bytecode": "0x608060405234801561001057600080fd5b50610e40806100206000396000f3fe608060405234801561001057600080fd5b5060043610610053576000357c01000000000000000000000000000000000000000000000000000000009004806344e8fd0814610058578063ac45fcd914610074575b600080fd5b610072600480360361006d91908101906108b4565b6100a4565b005b61008e60048036036100899190810190610920565b610319565b60405161009b9190610b9c565b60405180910390f35b6100ac610651565b6080604051908101604052803373ffffffffffffffffffffffffffffffffffffffff1681526020014281526020018481526020018381525090506000816040516020016100f99190610bfc565b604051602081830303815290604052805190602001209050816001600083815260200190815260200160002060008201518160000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550602082015181600101556040820151816002019080519060200190610192929190610690565b5060608201518160030190805190602001906101af929190610710565b50905050600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190806001815401808255809150509060018203906000526020600020016000909192909190915055506000829080600181540180825580915050906001820390600052602060002090600402016000909192909190915060008201518160000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506020820151816001015560408201518160020190805190602001906102b7929190610690565b5060608201518160030190805190602001906102d4929190610710565b505050507f4cbc6aabdd0942d8df984ae683445cc9d498eff032ced24070239d9a65603bb384823360405161030b93929190610bbe565b60405180910390a150505050565b606060008080549050141561036b57600060405190808252806020026020018201604052801561036357816020015b610350610790565b8152602001906001900390816103485790505b50905061064b565b600060018403830290506000816001600080549050030390506001600080549050038111156103d95760006040519080825280602002602001820160405280156103cf57816020015b6103bc610790565b8152602001906001900390816103b45790505b509250505061064b565b60008482039050818111156103ed57600090505b6000600182840301905085811115610403578590505b8060405190808252806020026020018201604052801561043d57816020015b61042a610790565b8152602001906001900390816104225790505b50945060008090505b8181101561064557600081850381548110151561045f57fe5b9060005260206000209060040201608060405190810160405290816000820160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200160018201548152602001600282018054600181600116156101000203166002900480601f0160208091040260200160405190810160405280929190818152602001828054600181600116156101000203166002900480156105725780601f1061054757610100808354040283529160200191610572565b820191906000526020600020905b81548152906001019060200180831161055557829003601f168201915b50505050508152602001600382018054600181600116156101000203166002900480601f0160208091040260200160405190810160405280929190818152602001828054600181600116156101000203166002900480156106145780601f106105e957610100808354040283529160200191610614565b820191906000526020600020905b8154815290600101906020018083116105f757829003601f168201915b505050505081525050868281518110151561062b57fe5b906020019060200201819052508080600101915050610446565b50505050505b92915050565b608060405190810160405280600073ffffffffffffffffffffffffffffffffffffffff1681526020016000815260200160608152602001606081525090565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106106d157805160ff19168380011785556106ff565b828001600101855582156106ff579182015b828111156106fe5782518255916020019190600101906106e3565b5b50905061070c91906107cf565b5090565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061075157805160ff191683800117855561077f565b8280016001018555821561077f579182015b8281111561077e578251825591602001919060010190610763565b5b50905061078c91906107cf565b5090565b608060405190810160405280600073ffffffffffffffffffffffffffffffffffffffff1681526020016000815260200160608152602001606081525090565b6107f191905b808211156107ed5760008160009055506001016107d5565b5090565b90565b600082601f830112151561080757600080fd5b813561081a61081582610c4b565b610c1e565b9150808252602083016020830185838301111561083657600080fd5b610841838284610db3565b50505092915050565b600082601f830112151561085d57600080fd5b813561087061086b82610c77565b610c1e565b9150808252602083016020830185838301111561088c57600080fd5b610897838284610db3565b50505092915050565b60006108ac8235610d73565b905092915050565b600080604083850312156108c757600080fd5b600083013567ffffffffffffffff8111156108e157600080fd5b6108ed8582860161084a565b925050602083013567ffffffffffffffff81111561090a57600080fd5b610916858286016107f4565b9150509250929050565b6000806040838503121561093357600080fd5b6000610941858286016108a0565b9250506020610952858286016108a0565b9150509250929050565b60006109688383610b23565b905092915050565b61097981610d7d565b82525050565b61098881610d2d565b82525050565b600061099982610cb0565b6109a38185610ce9565b9350836020820285016109b585610ca3565b60005b848110156109ee5783830388526109d083835161095c565b92506109db82610cdc565b91506020880197506001810190506109b8565b508196508694505050505092915050565b610a0881610d3f565b82525050565b6000610a1982610cbb565b610a238185610cfa565b9350610a33818560208601610dc2565b610a3c81610df5565b840191505092915050565b6000610a5282610cd1565b610a5c8185610d1c565b9350610a6c818560208601610dc2565b610a7581610df5565b840191505092915050565b6000610a8b82610cc6565b610a958185610d0b565b9350610aa5818560208601610dc2565b610aae81610df5565b840191505092915050565b6000608083016000830151610ad1600086018261097f565b506020830151610ae46020860182610b8d565b5060408301518482036040860152610afc8282610a80565b91505060608301518482036060860152610b168282610a0e565b9150508091505092915050565b6000608083016000830151610b3b600086018261097f565b506020830151610b4e6020860182610b8d565b5060408301518482036040860152610b668282610a80565b91505060608301518482036060860152610b808282610a0e565b9150508091505092915050565b610b9681610d69565b82525050565b60006020820190508181036000830152610bb6818461098e565b905092915050565b60006060820190508181036000830152610bd88186610a47565b9050610be760208301856109ff565b610bf46040830184610970565b949350505050565b60006020820190508181036000830152610c168184610ab9565b905092915050565b6000604051905081810181811067ffffffffffffffff82111715610c4157600080fd5b8060405250919050565b600067ffffffffffffffff821115610c6257600080fd5b601f19601f8301169050602081019050919050565b600067ffffffffffffffff821115610c8e57600080fd5b601f19601f8301169050602081019050919050565b6000602082019050919050565b600081519050919050565b600081519050919050565b600081519050919050565b600081519050919050565b6000602082019050919050565b600082825260208201905092915050565b600082825260208201905092915050565b600082825260208201905092915050565b600082825260208201905092915050565b6000610d3882610d49565b9050919050565b6000819050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000819050919050565b6000819050919050565b6000610d8882610d8f565b9050919050565b6000610d9a82610da1565b9050919050565b6000610dac82610d49565b9050919050565b82818337600083830152505050565b60005b83811015610de0578082015181840152602081019050610dc5565b83811115610def576000848401525b50505050565b6000601f19601f830116905091905056fea265627a7a72305820bc6fd206f999fbcfe9512092d362c4a7e7309d2bd608f91b9e3120086bea36646c6578706572696d656e74616cf50037",
	"source": "contract Registry {\n\n    struct message {\n        address sender;\n        uint timestamp;\n        string subject;\n        bytes hash;\n    }\n\n    message[] internal messages;\n    mapping(bytes32 => message) internal hashedMessages;\n    mapping(address => bytes32[]) internal senderMessages;\n\n    event Published(string subject, bytes32 key, address sender);\n\n    function publish(string memory _subject, bytes memory _hash) public {\n        message memory _msg = message(msg.sender, now, _subject, _hash);\n        bytes32 key = keccak256(abi.encode(_msg));\n        hashedMessages[key] = _msg;\n        senderMessages[msg.sender].push(key);\n        messages.push(_msg);\n        emit Published(_subject, key, msg.sender);\n    }\n\n    function listMessages(uint256 _page, uint256 _rpp) external view returns (message[] memory _msgs) {\n        if (messages.length == 0) {\n            return new message[](0);\n        }\n\n        uint256 _offset = _rpp * (_page - 1);\n        uint256 _index = messages.length - 1 - _offset;\n        if (_index > messages.length - 1) {\n            return new message[](0);\n        }\n\n        uint256 _lastIndex = _index - _rpp;\n        if (_lastIndex > _index) {\n            _lastIndex = 0;\n        }\n\n        uint256 _len = _index - _lastIndex + 1;\n        if (_len > _rpp) {\n            _len = _rpp;\n        }\n\n        _msgs = new message[](_len);\n        for (uint256 _i = 0; _i < _len; _i++) {\n            _msgs[_i] = messages[_index - _i];\n        }\n    }\n}"
}`

const ShieldArtifact = `{
	"contractName":"shield",
	"abi": [
		{
			"inputs": [
				{
					"internalType": "address",
					"name": "_verifier",
					"type": "address"
				},
				{
					"internalType": "uint256",
					"name": "_treeHeight",
					"type": "uint256"
				}
			],
			"stateMutability": "nonpayable",
			"type": "constructor"
		},
		{
			"anonymous": false,
			"inputs": [
				{
					"indexed": false,
					"internalType": "uint256",
					"name": "leafIndex",
					"type": "uint256"
				},
				{
					"indexed": false,
					"internalType": "bytes32",
					"name": "leafValue",
					"type": "bytes32"
				},
				{
					"indexed": false,
					"internalType": "bytes32",
					"name": "root",
					"type": "bytes32"
				}
			],
			"name": "NewLeaf",
			"type": "event"
		},
		{
			"anonymous": false,
			"inputs": [
				{
					"indexed": false,
					"internalType": "uint256",
					"name": "minLeafIndex",
					"type": "uint256"
				},
				{
					"indexed": false,
					"internalType": "bytes32[]",
					"name": "leafValues",
					"type": "bytes32[]"
				},
				{
					"indexed": false,
					"internalType": "bytes32",
					"name": "root",
					"type": "bytes32"
				}
			],
			"name": "NewLeaves",
			"type": "event"
		},
		{
			"anonymous": false,
			"inputs": [
				{
					"indexed": false,
					"internalType": "bytes27",
					"name": "leftInput",
					"type": "bytes27"
				},
				{
					"indexed": false,
					"internalType": "bytes27",
					"name": "rightInput",
					"type": "bytes27"
				},
				{
					"indexed": false,
					"internalType": "bytes32",
					"name": "output",
					"type": "bytes32"
				},
				{
					"indexed": false,
					"internalType": "uint256",
					"name": "nodeIndex",
					"type": "uint256"
				}
			],
			"name": "Output",
			"type": "event"
		},
		{
			"inputs": [
				{
					"internalType": "uint256",
					"name": "",
					"type": "uint256"
				}
			],
			"name": "frontier",
			"outputs": [
				{
					"internalType": "bytes27",
					"name": "",
					"type": "bytes27"
				}
			],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [],
			"name": "latestRoot",
			"outputs": [
				{
					"internalType": "bytes32",
					"name": "",
					"type": "bytes32"
				}
			],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [],
			"name": "leafCount",
			"outputs": [
				{
					"internalType": "uint256",
					"name": "",
					"type": "uint256"
				}
			],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [],
			"name": "treeHeight",
			"outputs": [
				{
					"internalType": "uint256",
					"name": "",
					"type": "uint256"
				}
			],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [],
			"name": "treeWidth",
			"outputs": [
				{
					"internalType": "uint256",
					"name": "",
					"type": "uint256"
				}
			],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [],
			"name": "zero",
			"outputs": [
				{
					"internalType": "bytes27",
					"name": "",
					"type": "bytes27"
				}
			],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [],
			"name": "getVerifier",
			"outputs": [
				{
					"internalType": "address",
					"name": "",
					"type": "address"
				}
			],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [
				{
					"internalType": "uint256[]",
					"name": "_proof",
					"type": "uint256[]"
				},
				{
					"internalType": "uint256[]",
					"name": "_publicInputs",
					"type": "uint256[]"
				},
				{
					"internalType": "bytes32",
					"name": "_newCommitment",
					"type": "bytes32"
				}
			],
			"name": "verifyAndPush",
			"outputs": [
				{
					"internalType": "bool",
					"name": "",
					"type": "bool"
				}
			],
			"stateMutability": "nonpayable",
			"type": "function"
		}
	],
	"bytecode": "0x608060405234801561001057600080fd5b50604051610d64380380610d648339818101604052606081101561003357600080fd5b810190808051604051939291908464010000000082111561005357600080fd5b8382019150602082018581111561006957600080fd5b825186600182028301116401000000008211171561008657600080fd5b8083526020830192505050908051906020019080838360005b838110156100ba57808201518184015260208101905061009f565b50505050905090810190601f1680156100e75780820380516001836020036101000a031916815260200191505b50604052602001805190602001909291908051906020019092919050505060008282604051610115906103c5565b808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200192505050604051809103906000f08015801561016e573d6000803e3d6000fd5b5090507f66e5424a1f01ba3b8a0748efe88f02b87554eb9d883f8e1c46d2e741f5084cef81604051808060200180602001806020018573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001848103845260198152602001807f73687574746c652e636f6e74726163742e6465706c6f79656400000000000000815250602001848103835260068152602001807f736869656c640000000000000000000000000000000000000000000000000000815250602001848103825260068152602001807f536869656c64000000000000000000000000000000000000000000000000000081525060200194505050505060405180910390a17f8a6056c5e7f38f90c2187227ca891f1e4db2d6d4783404f61d3f0e0930918fc88484836040518080602001806020018573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001838103835260188152602001807f73687574746c652e636972637569742e6465706c6f7965640000000000000000815250602001838103825286818151815260200191508051906020019080838360005b8381101561037f578082015181840152602081019050610364565b50505050905090810190601f1680156103ac5780820380516001836020036101000a031916815260200191505b509550505050505060405180910390a1505050506103d2565b6109458061041f83390190565b603f806103e06000396000f3fe6080604052600080fdfea264697066735822122048cfd26d1401e85ae56852a934004ba9519da5f6d8e86d77ec6f5fa52b86a04964736f6c634300060900336080604052600060281b600460006101000a8154817affffffffffffffffffffffffffffffffffffffffffffffffffffff021916908360281c021790555034801561004957600080fd5b506040516109453803806109458339818101604052604081101561006c57600080fd5b810190808051906020019092919080519060200190929190505050808060008190555060005460020a6001819055505081602660006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505050610857806100ee6000396000f3fe608060405234801561001057600080fd5b50600436106100885760003560e01c80636e0c3fee1161005b5780636e0c3fee1461020357806376c601b114610255578063bc1b392d14610273578063d7b0fef1146102a157610088565b806301e3e9151461008d57806330e69fc3146100ab57806346657fe9146100c95780636620e73b14610113575b600080fd5b6100956102bf565b6040518082815260200191505060405180910390f35b6100b36102c5565b6040518082815260200191505060405180910390f35b6100d16102cb565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6101e96004803603606081101561012957600080fd5b810190808035906020019064010000000081111561014657600080fd5b82018360208201111561015857600080fd5b8035906020019184602083028401116401000000008311171561017a57600080fd5b90919293919293908035906020019064010000000081111561019b57600080fd5b8201836020820111156101ad57600080fd5b803590602001918460208302840111640100000000831117156101cf57600080fd5b9091929391929390803590602001909291905050506102f5565b604051808215151515815260200191505060405180910390f35b61022f6004803603602081101561021957600080fd5b810190808035906020019092919050505061047e565b604051808264ffffffffff191664ffffffffff1916815260200191505060405180910390f35b61025d61049e565b6040518082815260200191505060405180910390f35b61027b6104a4565b604051808264ffffffffff191664ffffffffff1916815260200191505060405180910390f35b6102a96104b7565b6040518082815260200191505060405180910390f35b60005481565b60035481565b6000602660009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b600080602660009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663b864f5a9888888886040518563ffffffff1660e01b81526004018080602001806020018381038352878782818152602001925060200280828437600081840152601f19601f8201169050808301925050508381038252858582818152602001925060200280828437600081840152601f19601f8201169050808301925050509650505050505050602060405180830381600087803b1580156103d357600080fd5b505af11580156103e7573d6000803e3d6000fd5b505050506040513d60208110156103fd57600080fd5b8101908080519060200190929190505050905080610466576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260368152602001806107c96036913960400191505060405180910390fd5b61046f836104bd565b50600191505095945050505050565b6005816021811061048b57fe5b016000915054906101000a900460281b81565b60015481565b600460009054906101000a900460281b81565b60025481565b60006003546001541161051b576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260238152602001806107ff6023913960400191505060405180910390fd5b600061052860035461072f565b9050600060018054600354010390506000602885901b905060008061054b6107a6565b600080600090505b6000548110156106a857878114156105a757856005896021811061057357fe5b0160006101000a8154817affffffffffffffffffffffffffffffffffffffffffffffffffffff021916908360281c02179055505b6000600288816105b357fe5b06141561063457600581602181106105c757fe5b0160009054906101000a900460281b945085935060405185815284601b8201526020846036836002600019fa9250826000811461060357610605565bfe5b505060288360006001811061061657fe5b6020020151901b95506002600188038161062c57fe5b04965061069b565b859450600460009054906101000a900460281b935060405185815284601b8201526020846036836002600019fa9250826000811461067157610673565bfe5b505060288360006001811061068457fe5b6020020151901b95506002878161069757fe5b0496505b8080600101915050610553565b50816000600181106106b657fe5b60200201516002819055507f6a82ba2aa1d2c039c41e6e2b5a5a1090d09906f060d32af9c1ac0beff7af75c06003548a60025460405180848152602001838152602001828152602001935050505060405180910390a1600360008154809291906001019190505550600254975050505050505050919050565b600080905060016002838161074057fe5b0614156107a1576000600190506000600290506000600182901b90505b600084141561079d576000818360018801038161077657fe5b06141561078557829350610798565b809150600181901b905082806001019350505b61075d565b5050505b919050565b604051806020016040528060019060208202803683378082019150509050509056fe5468652070726f6f66206661696c656420766572696669636174696f6e20696e2074686520766572696669657220636f6e74726163745468657265206973206e6f207370616365206c65667420696e2074686520747265652ea26469706673582212205b68cc9ccdc7b7ef2e92fabde0dfd489efa619e88eaf2f700bf2f3f0731aff7964736f6c63430006090033",
	"source": "pragma solidity ^0.6.9;\n\nimport \"./privacy/Shield.sol\";\n\ncontract ShuttleCircuit {\n\n    event ShuttleDeployedContract(string subject, string contractType, string name, address addr);\n    event ShuttleDeployedCircuit(string subject, string name, address verifier, address shield);\n\n    constructor(string memory name, address verifier, uint treeHeight) public {\n        Shield shield = new Shield(verifier, treeHeight);\n        emit ShuttleDeployedContract(\"shuttle.contract.deployed\", \"shield\", \"Shield\", address(shield));\n        emit ShuttleDeployedCircuit(\"shuttle.circuit.deployed\", name, address(verifier), address(shield));\n    }\n}\n"
}`

const ERC1820RegistryArtifact = `{
	"contractName":"ERC1820Registry",
	"abi": [
		{
			"constant": true,
			"inputs": [
				{
					"name": "interfaceId",
					"type": "bytes4"
				}
			],
			"name": "supportsInterface",
			"outputs": [
				{
					"name": "",
					"type": "bool"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": [
				{
					"name": "_address",
					"type": "address"
				}
			],
			"name": "getOrg",
			"outputs": [
				{
					"name": "",
					"type": "address"
				},
				{
					"name": "",
					"type": "bytes32"
				},
				{
					"name": "",
					"type": "bytes"
				},
				{
					"name": "",
					"type": "bytes"
				},
				{
					"name": "",
					"type": "bytes"
				},
				{
					"name": "",
					"type": "bytes"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": null,
			"name": "getOrgCount",
			"outputs": [
				{
					"name": "",
					"type": "uint256"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": [
				{
					"name": "interfaceHash",
					"type": "bytes32"
				},
				{
					"name": "addr",
					"type": "address"
				}
			],
			"name": "canImplementInterfaceForAddress",
			"outputs": [
				{
					"name": "",
					"type": "bytes32"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": false,
			"inputs": [
				{
					"name": "_newManager",
					"type": "address"
				}
			],
			"name": "assignManager",
			"outputs": null,
			"payable": false,
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": [
				{
					"name": "addr",
					"type": "address"
				},
				{
					"name": "_interfaceLabel",
					"type": "string"
				}
			],
			"name": "interfaceAddr",
			"outputs": [
				{
					"name": "",
					"type": "address"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": false,
			"inputs": [
				{
					"name": "_address",
					"type": "address"
				},
				{
					"name": "_name",
					"type": "bytes32"
				},
				{
					"name": "_messengerEndpoint",
					"type": "bytes"
				},
				{
					"name": "_whisperKey",
					"type": "bytes"
				},
				{
					"name": "_zkpPublicKey",
					"type": "bytes"
				},
				{
					"name": "_metadata",
					"type": "bytes"
				}
			],
			"name": "registerOrg",
			"outputs": [
				{
					"name": "",
					"type": "bool"
				}
			],
			"payable": false,
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"constant": false,
			"inputs": [
				{
					"name": "_groupName",
					"type": "bytes32"
				},
				{
					"name": "_tokenAddress",
					"type": "address"
				},
				{
					"name": "_shieldAddress",
					"type": "address"
				},
				{
					"name": "_verifierAddress",
					"type": "address"
				}
			],
			"name": "registerInterfaces",
			"outputs": [
				{
					"name": "",
					"type": "bool"
				}
			],
			"payable": false,
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": null,
			"name": "owner",
			"outputs": [
				{
					"name": "",
					"type": "address"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": null,
			"name": "_owner",
			"outputs": [
				{
					"name": "",
					"type": "address"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": null,
			"name": "getManager",
			"outputs": [
				{
					"name": "",
					"type": "address"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": null,
			"name": "getInterfaceAddresses",
			"outputs": [
				{
					"name": "",
					"type": "bytes32[]"
				},
				{
					"name": "",
					"type": "address[]"
				},
				{
					"name": "",
					"type": "address[]"
				},
				{
					"name": "",
					"type": "address[]"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": null,
			"name": "getInterfaces",
			"outputs": [
				{
					"name": "",
					"type": "bytes4"
				}
			],
			"payable": false,
			"stateMutability": "pure",
			"type": "function"
		},
		{
			"constant": false,
			"inputs": [
				{
					"name": "newOwner",
					"type": "address"
				}
			],
			"name": "transferOwnership",
			"outputs": null,
			"payable": false,
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"constant": false,
			"inputs": null,
			"name": "setInterfaces",
			"outputs": [
				{
					"name": "",
					"type": "bool"
				}
			],
			"payable": false,
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"inputs": [
				{
					"name": "_erc1820",
					"type": "address"
				}
			],
			"payable": false,
			"stateMutability": "nonpayable",
			"type": "constructor"
		},
		{
			"anonymous": false,
			"inputs": [
				{
					"indexed": false,
					"name": "_name",
					"type": "bytes32"
				},
				{
					"indexed": false,
					"name": "_address",
					"type": "address"
				},
				{
					"indexed": false,
					"name": "_messengerEndpoint",
					"type": "bytes"
				},
				{
					"indexed": false,
					"name": "_whisperKey",
					"type": "bytes"
				},
				{
					"indexed": false,
					"name": "_zkpPublicKey",
					"type": "bytes"
				},
				{
					"indexed": false,
					"name": "_metadata",
					"type": "bytes"
				}
			],
			"name": "RegisterOrg",
			"type": "event"
		},
		{
			"anonymous": false,
			"inputs": [
				{
					"indexed": false,
					"name": "previousOwner",
					"type": "address"
				},
				{
					"indexed": false,
					"name": "newOwner",
					"type": "address"
				}
			],
			"name": "OwnershipTransferred",
			"type": "event"
		}
	],
	"bytecode": "0x608060405234801561001057600080fd5b5061128a806100206000396000f3fe608060405234801561001057600080fd5b50600436106100885760003560e01c8063a41e7d511161005b578063a41e7d5114610125578063aabbb8ca14610141578063b705676514610171578063f712f3e8146101a157610088565b806329965a1d1461008d5780633d584063146100a95780635df8122f146100d957806365ba36c1146100f5575b600080fd5b6100a760048036036100a29190810190610e10565b6101d1565b005b6100c360048036036100be9190810190610d6f565b61051d565b6040516100d09190611089565b60405180910390f35b6100f360048036036100ee9190810190610d98565b610622565b005b61010f600480360361010a9190810190610ec4565b6107af565b60405161011c91906110bf565b60405180910390f35b61013f600480360361013a9190810190610e5f565b6107e2565b005b61015b60048036036101569190810190610dd4565b610950565b6040516101689190611089565b60405180910390f35b61018b60048036036101869190810190610e5f565b610a3e565b60405161019891906110a4565b60405180910390f35b6101bb60048036036101b69190810190610e5f565b610af3565b6040516101c891906110a4565b60405180910390f35b60008073ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff161461020c578361020e565b335b90503373ffffffffffffffffffffffffffffffffffffffff166102308261051d565b73ffffffffffffffffffffffffffffffffffffffff1614610286576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161027d90611103565b60405180910390fd5b61028f83610c6c565b156102cf576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016102c690611123565b60405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff161415801561033857503373ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1614155b1561042e5760405160200161034c90611074565b604051602081830303815290604052805190602001208273ffffffffffffffffffffffffffffffffffffffff1663249cb3fa85846040518363ffffffff1660e01b815260040161039d9291906110da565b60206040518083038186803b1580156103b557600080fd5b505afa1580156103c9573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052506103ed9190810190610e9b565b1461042d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161042490611143565b60405180910390fd5b5b816000808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600085815260200190815260200160002060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508173ffffffffffffffffffffffffffffffffffffffff16838273ffffffffffffffffffffffffffffffffffffffff167f93baa6efbd2244243bfee6ce4cfdd1d04fc4c0e9a786abd3a41313bd352db15360405160405180910390a450505050565b60008073ffffffffffffffffffffffffffffffffffffffff16600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614156105ba5781905061061d565b600160008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1690505b919050565b3373ffffffffffffffffffffffffffffffffffffffff166106428361051d565b73ffffffffffffffffffffffffffffffffffffffff1614610698576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161068f90611103565b60405180910390fd5b8173ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16146106d157806106d4565b60005b600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff167f605c2dbf762e5f7d60a546d42e7205dcb1b011ebc62a61736a57c9089d3a435060405160405180910390a35050565b600082826040516020016107c492919061105b565b60405160208183030381529060405280519060200120905092915050565b6107ec8282610a3e565b6107f75760006107f9565b815b6000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000837bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260200190815260200160002060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506001600260008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000837bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19167bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260200190815260200160002060006101000a81548160ff0219169083151502179055505050565b600080600073ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff161461098d578361098f565b335b905061099a83610c6c565b156109c45760008390506109ae8282610af3565b6109b95760006109bb565b815b92505050610a38565b6000808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600084815260200190815260200160002060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff169150505b92915050565b6000806000610a54856301ffc9a760e01b610c9c565b80925081935050506000821480610a6b5750600081145b15610a7b57600092505050610aed565b610a8c8563ffffffff60e01b610c9c565b80925081935050506000821480610aa4575060008114155b15610ab457600092505050610aed565b610abe8585610c9c565b8092508193505050600182148015610ad65750600181145b15610ae657600192505050610aed565b6000925050505b92915050565b6000600260008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000837bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19167bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260200190815260200160002060009054906101000a900460ff16610ba657610b9f8383610a3e565b9050610c66565b8273ffffffffffffffffffffffffffffffffffffffff166000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000847bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260200190815260200160002060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff161490505b92915050565b60008060001b7bffffffffffffffffffffffffffffffffffffffffffffffffffffffff60001b8316149050919050565b60008060006301ffc9a760e01b905060405181815284600482015260208160248389617530fa93508051925050509250929050565b600081359050610ce081611202565b92915050565b600081359050610cf581611219565b92915050565b600081519050610d0a81611219565b92915050565b600081359050610d1f81611230565b92915050565b60008083601f840112610d3757600080fd5b8235905067ffffffffffffffff811115610d5057600080fd5b602083019150836001820283011115610d6857600080fd5b9250929050565b600060208284031215610d8157600080fd5b6000610d8f84828501610cd1565b91505092915050565b60008060408385031215610dab57600080fd5b6000610db985828601610cd1565b9250506020610dca85828601610cd1565b9150509250929050565b60008060408385031215610de757600080fd5b6000610df585828601610cd1565b9250506020610e0685828601610ce6565b9150509250929050565b600080600060608486031215610e2557600080fd5b6000610e3386828701610cd1565b9350506020610e4486828701610ce6565b9250506040610e5586828701610cd1565b9150509250925092565b60008060408385031215610e7257600080fd5b6000610e8085828601610cd1565b9250506020610e9185828601610d10565b9150509250929050565b600060208284031215610ead57600080fd5b6000610ebb84828501610cfb565b91505092915050565b60008060208385031215610ed757600080fd5b600083013567ffffffffffffffff811115610ef157600080fd5b610efd85828601610d25565b92509250509250929050565b610f128161117f565b82525050565b610f2181611191565b82525050565b610f308161119d565b82525050565b6000610f428385611174565b9350610f4f8385846111f3565b82840190509392505050565b6000610f68600f83611163565b91507f4e6f7420746865206d616e6167657200000000000000000000000000000000006000830152602082019050919050565b6000610fa8601a83611163565b91507f4d757374206e6f7420626520616e2045524331363520686173680000000000006000830152602082019050919050565b6000610fe8601483611174565b91507f455243313832305f4143434550545f4d414749430000000000000000000000006000830152601482019050919050565b6000611028602083611163565b91507f446f6573206e6f7420696d706c656d656e742074686520696e746572666163656000830152602082019050919050565b6000611068828486610f36565b91508190509392505050565b600061107f82610fdb565b9150819050919050565b600060208201905061109e6000830184610f09565b92915050565b60006020820190506110b96000830184610f18565b92915050565b60006020820190506110d46000830184610f27565b92915050565b60006040820190506110ef6000830185610f27565b6110fc6020830184610f09565b9392505050565b6000602082019050818103600083015261111c81610f5b565b9050919050565b6000602082019050818103600083015261113c81610f9b565b9050919050565b6000602082019050818103600083015261115c8161101b565b9050919050565b600082825260208201905092915050565b600081905092915050565b600061118a826111d3565b9050919050565b60008115159050919050565b6000819050919050565b60007fffffffff0000000000000000000000000000000000000000000000000000000082169050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b82818337600083830152505050565b61120b8161117f565b811461121657600080fd5b50565b6112228161119d565b811461122d57600080fd5b50565b611239816111a7565b811461124457600080fd5b5056fea365627a7a723058205289db890632bfcfb834b8d41eaf07ccb0fd2d3cc680c061dbdd7e239cf643aa6c6578706572696d656e74616cf564736f6c63430005090040",
	"source": "'newManager' is the address of the new manager for 'addr'.\n    event ManagerChanged(address indexed addr, address indexed newManager);\n\n    /// @notice Query if an address implements an interface and through which contract.\n    /// @param _addr Address being queried for the implementer of an interface.\n    /// (If '_addr' is the zero address then 'msg.sender' is assumed.)\n    /// @param _interfaceHash Keccak256 hash of the name of the interface as a string.\n    /// E.g., 'web3.utils.keccak256(\"ERC777TokensRecipient\")' for the 'ERC777TokensRecipient' interface.\n    /// @return The address of the contract which implements the interface '_interfaceHash' for '_addr'\n    /// or '0' if '_addr' did not register an implementer for this interface.\n    function getInterfaceImplementer(address _addr, bytes32 _interfaceHash) external view returns (address) {\n        address addr = _addr == address(0) ? msg.sender : _addr;\n        if (isERC165Interface(_interfaceHash)) {\n            bytes4 erc165InterfaceHash = bytes4(_interfaceHash);\n            return implementsERC165Interface(addr, erc165InterfaceHash) ? addr : address(0);\n        }\n        return interfaces[addr][_interfaceHash];\n    }\n\n    /// @notice Sets the contract which implements a specific interface for an address.\n    /// Only the manager defined for that address can set it.\n    /// (Each address is the manager for itself until it sets a new manager.)\n    /// @param _addr Address for which to set the interface.\n    /// (If '_addr' is the zero address then 'msg.sender' is assumed.)\n    /// @param _interfaceHash Keccak256 hash of the name of the interface as a string.\n    /// E.g., 'web3.utils.keccak256(\"ERC777TokensRecipient\")' for the 'ERC777TokensRecipient' interface.\n    /// @param _implementer Contract address implementing '_interfaceHash' for '_addr'.\n    function setInterfaceImplementer(address _addr, bytes32 _interfaceHash, address _implementer) external {\n        address addr = _addr == address(0) ? msg.sender : _addr;\n        require(getManager(addr) == msg.sender, \"Not the manager\");\n\n        require(!isERC165Interface(_interfaceHash), \"Must not be an ERC165 hash\");\n        if (_implementer != address(0) && _implementer != msg.sender) {\n            require(\n                ERC1820ImplementerInterface(_implementer)\n                    .canImplementInterfaceForAddress(_interfaceHash, addr) == ERC1820_ACCEPT_MAGIC,\n                \"Does not implement the interface\"\n            );\n        }\n        interfaces[addr][_interfaceHash] = _implementer;\n        emit InterfaceImplementerSet(addr, _interfaceHash, _implementer);\n    }\n\n    /// @notice Sets '_newManager' as manager for '_addr'.\n    /// The new manager will be able to call 'setInterfaceImplementer' for '_addr'.\n    /// @param _addr Address for which to set the new manager.\n    /// @param _newManager Address of the new manager for 'addr'. (Pass '0x0' to reset the manager to '_addr'.)\n    function setManager(address _addr, address _newManager) external {\n        require(getManager(_addr) == msg.sender, \"Not the manager\");\n        managers[_addr] = _newManager == _addr ? address(0) : _newManager;\n        emit ManagerChanged(_addr, _newManager);\n    }\n\n    /// @notice Get the manager of an address.\n    /// @param _addr Address for which to return the manager.\n    /// @return Address of the manager for a given address.\n    function getManager(address _addr) public view returns(address) {\n        // By default the manager of an address is the same address\n        if (managers[_addr] == address(0)) {\n            return _addr;\n        } else {\n            return managers[_addr];\n        }\n    }\n\n    /// @notice Compute the keccak256 hash of an interface given its name.\n    /// @param _interfaceName Name of the interface.\n    /// @return The keccak256 hash of an interface name.\n    function interfaceHash(string calldata _interfaceName) external pure returns(bytes32) {\n        return keccak256(abi.encodePacked(_interfaceName));\n    }\n\n    /* --- ERC165 Related Functions --- */\n    /* --- Developed in collaboration with William Entriken. --- */\n\n    /// @notice Updates the cache with whether the contract implements an ERC165 interface or not.\n    /// @param _contract Address of the contract for which to update the cache.\n    /// @param _interfaceId ERC165 interface for which to update the cache.\n    function updateERC165Cache(address _contract, bytes4 _interfaceId) external {\n        interfaces[_contract][_interfaceId] = implementsERC165InterfaceNoCache(\n            _contract, _interfaceId) ? _contract : address(0);\n        erc165Cached[_contract][_interfaceId] = true;\n    }\n\n    /// @notice Checks whether a contract implements an ERC165 interface or not.\n    //  If the result is not cached a direct lookup on the contract address is performed.\n    //  If the result is not cached or the cached value is out-of-date, the cache MUST be updated manually by calling\n    //  'updateERC165Cache' with the contract address.\n    /// @param _contract Address of the contract to check.\n    /// @param _interfaceId ERC165 interface to check.\n    /// @return True if '_contract' implements '_interfaceId', false otherwise.\n    function implementsERC165Interface(address _contract, bytes4 _interfaceId) public view returns (bool) {\n        if (!erc165Cached[_contract][_interfaceId]) {\n            return implementsERC165InterfaceNoCache(_contract, _interfaceId);\n        }\n        return interfaces[_contract][_interfaceId] == _contract;\n    }\n\n    /// @notice Checks whether a contract implements an ERC165 interface or not without using nor updating the cache.\n    /// @param _contract Address of the contract to check.\n    /// @param _interfaceId ERC165 interface to check.\n    /// @return True if '_contract' implements '_interfaceId', false otherwise.\n    function implementsERC165InterfaceNoCache(address _contract, bytes4 _interfaceId) public view returns (bool) {\n        uint256 success;\n        uint256 result;\n\n        (success, result) = noThrowCall(_contract, ERC165ID);\n        if (success == 0 || result == 0) {\n            return false;\n        }\n\n        (success, result) = noThrowCall(_contract, INVALID_ID);\n        if (success == 0 || result != 0) {\n            return false;\n        }\n\n        (success, result) = noThrowCall(_contract, _interfaceId);\n        if (success == 1 && result == 1) {\n            return true;\n        }\n        return false;\n    }\n\n    /// @notice Checks whether the hash is a ERC165 interface (ending with 28 zeroes) or not.\n    /// @param _interfaceHash The hash to check.\n    /// @return True if '_interfaceHash' is an ERC165 interface (ending with 28 zeroes), false otherwise.\n    function isERC165Interface(bytes32 _interfaceHash) internal pure returns (bool) {\n        return _interfaceHash & 0x00000000FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF == 0;\n    }\n\n    /// @dev Make a call on a contract without throwing if the function does not exist.\n    function noThrowCall(address _contract, bytes4 _interfaceId)\n        internal view returns (uint256 success, uint256 result)\n    {\n        bytes4 erc165ID = ERC165ID;\n\n        assembly {\n            let x := mload(0x40)               // Find empty storage location using \"free memory pointer\"\n            mstore(x, erc165ID)                // Place signature at beginning of empty storage\n            mstore(add(x, 0x04), _interfaceId) // Place first argument directly next to signature\n\n            success := staticcall(\n                30000,                         // 30k gas\n                _contract,                     // To addr\n                x,                             // Inputs are stored at location x\n                0x24,                          // Inputs are 36 (4 + 32) bytes long\n                x,                             // Store output over input (saves space)\n                0x20                           // Outputs are 32 bytes long\n            )\n\n            result := mload(x)                 // Load the result\n        }\n    }\n}\n\n\n/// @dev Contract that acts as a client for interacting with the ERC1820Registry\ncontract Registrar {\n\n    ERC1820Registry ERC1820REGISTRY;\n\n    bytes32 constant internal ERC1820_ACCEPT_MAGIC = keccak256(abi.encodePacked(\"ERC1820_ACCEPT_MAGIC\"));\n\n    /**\n    * @dev Throws if called by any account other than the owner.\n    */\n    modifier onlyManager() {\n        require(msg.sender == getManager(), \"You are not authorised to invoke this function\");\n        _;\n    }\n\n\n    /// @notice Constructor that takes an argument of the ERC1820RegistryAddress\n    /// @dev Upon actual deployment of a static registry contract, this argument can be removed\n    /// @param ERC1820RegistryAddress pre-deployed ERC1820 registry address\n    constructor (address ERC1820RegistryAddress) public {\n        // Below line is to be uncommented during actual deployment since mainnet has a version of this address\n        // ERC1820Registry constant ERC1820REGISTRY = ERC1820Registry(0x1820a4B7618BdE71Dce8cdc73aAB6C95905faD24);\n        ERC1820REGISTRY = ERC1820Registry(ERC1820RegistryAddress);\n    }\n\n    /// @dev This enables setting the interface implementation\n    /// @notice Sin"
}`

// const RegistryArtifact = `{
// 	"contractName":"xxx",
// 	"abi": ,
// 	"bytecode": "xxx",
// 	"source": "xxx"
// }`

const greeterArtifact = `{
  "contractName": "Greeter",
  "abi": [
    {
      "constant": false,
      "inputs": [],
      "name": "constuctor",
      "outputs": [],
      "payable": false,
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "greet",
      "outputs": [
        {
          "internalType": "string",
          "name": "",
          "type": "string"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "getBlockNumber",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": false,
      "inputs": [
        {
          "internalType": "string",
          "name": "_newgreeting",
          "type": "string"
        }
      ],
      "name": "setGreeting",
      "outputs": [],
      "payable": false,
      "stateMutability": "nonpayable",
      "type": "function"
    }
  ],
  "metadata": "{\"compiler\":{\"version\":\"0.5.16+commit.9c3226ce\"},\"language\":\"Solidity\",\"output\":{\"abi\":[{\"constant\":false,\"inputs\":[],\"name\":\"constuctor\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"greet\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"string\",\"name\":\"_newgreeting\",\"type\":\"string\"}],\"name\":\"setGreeting\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}],\"devdoc\":{\"methods\":{}},\"userdoc\":{\"methods\":{}}},\"settings\":{\"compilationTarget\":{\"/home/eoin/projects/provide/truffle/contracts/greeter.sol\":\"Greeter\"},\"evmVersion\":\"istanbul\",\"libraries\":{},\"optimizer\":{\"enabled\":false,\"runs\":200},\"remappings\":[]},\"sources\":{\"/home/eoin/projects/provide/truffle/contracts/greeter.sol\":{\"keccak256\":\"0x4d35ace6a7e1a453c4bd90d18505b375cdafa01ca59391e8cf81109d00f6ecaf\",\"urls\":[\"bzz-raw://929e3de323fc90f0ac5d7d4f1bf6518bdff93a49b0d05c691b320e8f5596df5d\",\"dweb:/ipfs/QmQ6oYEhnCFWg7DbZSqU6jXo38q9TGf2kM5mojcmooqiDa\"]}},\"version\":1}",
  "bytecode": "0x608060405234801561001057600080fd5b5061037e806100206000396000f3fe608060405234801561001057600080fd5b506004361061004c5760003560e01c806342cbb15c14610051578063a41368621461006f578063cfae32171461012a578063dfe4858a146101ad575b600080fd5b6100596101b7565b6040518082815260200191505060405180910390f35b6101286004803603602081101561008557600080fd5b81019080803590602001906401000000008111156100a257600080fd5b8201836020820111156100b457600080fd5b803590602001918460018302840111640100000000831117156100d657600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f8201169050808301925050505050505091929192905050506101bf565b005b6101326101d9565b6040518080602001828103825283818151815260200191508051906020019080838360005b83811015610172578082015181840152602081019050610157565b50505050905090810190601f16801561019f5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b6101b5610216565b005b600043905090565b80600190805190602001906101d59291906102a4565b5050565b60606040518060400160405280600b81526020017f68656c6c6f20776f726c64000000000000000000000000000000000000000000815250905090565b336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506040518060400160405280600b81526020017f68656c6c6f20776f726c64000000000000000000000000000000000000000000815250600190805190602001906102a19291906102a4565b50565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106102e557805160ff1916838001178555610313565b82800160010185558215610313579182015b828111156103125782518255916020019190600101906102f7565b5b5090506103209190610324565b5090565b61034691905b8082111561034257600081600090555060010161032a565b5090565b9056fea265627a7a72315820a2797318982ecebee88c185526f1ac5b1b24689a1968199b36d03d03c16ffd9764736f6c63430005100032",
  "deployedBytecode": "0x608060405234801561001057600080fd5b506004361061004c5760003560e01c806342cbb15c14610051578063a41368621461006f578063cfae32171461012a578063dfe4858a146101ad575b600080fd5b6100596101b7565b6040518082815260200191505060405180910390f35b6101286004803603602081101561008557600080fd5b81019080803590602001906401000000008111156100a257600080fd5b8201836020820111156100b457600080fd5b803590602001918460018302840111640100000000831117156100d657600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f8201169050808301925050505050505091929192905050506101bf565b005b6101326101d9565b6040518080602001828103825283818151815260200191508051906020019080838360005b83811015610172578082015181840152602081019050610157565b50505050905090810190601f16801561019f5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b6101b5610216565b005b600043905090565b80600190805190602001906101d59291906102a4565b5050565b60606040518060400160405280600b81526020017f68656c6c6f20776f726c64000000000000000000000000000000000000000000815250905090565b336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506040518060400160405280600b81526020017f68656c6c6f20776f726c64000000000000000000000000000000000000000000815250600190805190602001906102a19291906102a4565b50565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106102e557805160ff1916838001178555610313565b82800160010185558215610313579182015b828111156103125782518255916020019190600101906102f7565b5b5090506103209190610324565b5090565b61034691905b8082111561034257600081600090555060010161032a565b5090565b9056fea265627a7a72315820a2797318982ecebee88c185526f1ac5b1b24689a1968199b36d03d03c16ffd9764736f6c63430005100032",
  "sourceMap": "334:911:0:-;;;;8:9:-1;5:2;;;30:1;27;20:12;5:2;334:911:0;;;;;;;",
  "deployedSourceMap": "334:911:0:-;;;;8:9:-1;5:2;;;30:1;27;20:12;5:2;334:911:0;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;1052:89;;;:::i;:::-;;;;;;;;;;;;;;;;;;;1147:96;;;;;;13:2:-1;8:3;5:11;2:2;;;29:1;26;19:12;2:2;1147:96:0;;;;;;;;;;21:11:-1;8;5:28;2:2;;;46:1;43;36:12;2:2;1147:96:0;;35:9:-1;28:4;12:14;8:25;5:40;2:2;;;58:1;55;48:12;2:2;1147:96:0;;;;;;100:9:-1;95:1;81:12;77:20;67:8;63:35;60:50;39:11;25:12;22:29;11:107;8:2;;;131:1;128;121:12;8:2;1147:96:0;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;30:3:-1;22:6;14;1:33;99:1;93:3;85:6;81:16;74:27;137:4;133:9;126:4;121:3;117:14;113:30;106:37;;169:3;161:6;157:16;147:26;;1147:96:0;;;;;;;;;;;;;;;:::i;:::-;;834:90;;;:::i;:::-;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;23:1:-1;8:100;33:3;30:1;27:10;8:100;;;99:1;94:3;90:11;84:18;80:1;75:3;71:11;64:39;52:2;49:1;45:10;40:15;;8:100;;;12:14;834:90:0;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;728:100;;;:::i;:::-;;1052:89;1099:4;1122:12;1115:19;;1052:89;:::o;1147:96::-;1224:12;1213:8;:23;;;;;;;;;;;;:::i;:::-;;1147:96;:::o;834:90::-;872:13;897:20;;;;;;;;;;;;;;;;;;;834:90;:::o;728:100::-;777:10;767:7;;:20;;;;;;;;;;;;;;;;;;797:24;;;;;;;;;;;;;;;;;:8;:24;;;;;;;;;;;;:::i;:::-;;728:100::o;334:911::-;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;:::i;:::-;;;:::o;:::-;;;;;;;;;;;;;;;;;;;;;;;;;;;:::o",
  "source": "/*\n\tThe following is an extremely basic example of a solidity contract.\n\tIt takes a string upon creation and then repeats it when greet() is called.\n*/\n\n/// @title Greeter\n/// @author Cyrus Adkisson\n// The contract definition. A constructor of the same name will be automatically called on contract creation.\npragma solidity ^0.5.16;\ncontract Greeter {\n\n    // At first, an empty \"address\"-type variable of the name \"creator\". Will be set in the constructor.\n    address creator;\n    // At first, an empty \"string\"-type variable of the name \"greeting\". Will be set in constructor and can be changed.\n    string greeting;\n\n    // The constructor. It accepts a string input and saves it to the contract's \"greeting\" variable.\n    function constuctor() public {\n        creator = msg.sender;\n        greeting = 'hello world';\n    }\n\n    function greet() public view returns (string memory) {\n        return 'hello world';\n    }\n\n    // this doesn't have anything to do with the act of greeting\n    // just demonstrating return of some global variable\n    function getBlockNumber() public view returns (uint) {\n        return block.number;\n    }\n\n    function setGreeting(string memory _newgreeting) public {\n        greeting = _newgreeting;\n    }\n}\n",
  "sourcePath": "/home/eoin/projects/provide/truffle/contracts/greeter.sol",
  "ast": {
    "absolutePath": "/home/eoin/projects/provide/truffle/contracts/greeter.sol",
    "exportedSymbols": {
      "Greeter": [
        46
      ]
    },
    "id": 47,
    "nodeType": "SourceUnit",
    "nodes": [
      {
        "id": 1,
        "literals": [
          "solidity",
          "^",
          "0.5",
          ".16"
        ],
        "nodeType": "PragmaDirective",
        "src": "309:24:0"
      },
      {
        "baseContracts": [],
        "contractDependencies": [],
        "contractKind": "contract",
        "documentation": null,
        "fullyImplemented": true,
        "id": 46,
        "linearizedBaseContracts": [
          46
        ],
        "name": "Greeter",
        "nodeType": "ContractDefinition",
        "nodes": [
          {
            "constant": false,
            "id": 3,
            "name": "creator",
            "nodeType": "VariableDeclaration",
            "scope": 46,
            "src": "463:15:0",
            "stateVariable": true,
            "storageLocation": "default",
            "typeDescriptions": {
              "typeIdentifier": "t_address",
              "typeString": "address"
            },
            "typeName": {
              "id": 2,
              "name": "address",
              "nodeType": "ElementaryTypeName",
              "src": "463:7:0",
              "stateMutability": "nonpayable",
              "typeDescriptions": {
                "typeIdentifier": "t_address",
                "typeString": "address"
              }
            },
            "value": null,
            "visibility": "internal"
          },
          {
            "constant": false,
            "id": 5,
            "name": "greeting",
            "nodeType": "VariableDeclaration",
            "scope": 46,
            "src": "604:15:0",
            "stateVariable": true,
            "storageLocation": "default",
            "typeDescriptions": {
              "typeIdentifier": "t_string_storage",
              "typeString": "string"
            },
            "typeName": {
              "id": 4,
              "name": "string",
              "nodeType": "ElementaryTypeName",
              "src": "604:6:0",
              "typeDescriptions": {
                "typeIdentifier": "t_string_storage_ptr",
                "typeString": "string"
              }
            },
            "value": null,
            "visibility": "internal"
          },
          {
            "body": {
              "id": 17,
              "nodeType": "Block",
              "src": "757:71:0",
              "statements": [
                {
                  "expression": {
                    "argumentTypes": null,
                    "id": 11,
                    "isConstant": false,
                    "isLValue": false,
                    "isPure": false,
                    "lValueRequested": false,
                    "leftHandSide": {
                      "argumentTypes": null,
                      "id": 8,
                      "name": "creator",
                      "nodeType": "Identifier",
                      "overloadedDeclarations": [],
                      "referencedDeclaration": 3,
                      "src": "767:7:0",
                      "typeDescriptions": {
                        "typeIdentifier": "t_address",
                        "typeString": "address"
                      }
                    },
                    "nodeType": "Assignment",
                    "operator": "=",
                    "rightHandSide": {
                      "argumentTypes": null,
                      "expression": {
                        "argumentTypes": null,
                        "id": 9,
                        "name": "msg",
                        "nodeType": "Identifier",
                        "overloadedDeclarations": [],
                        "referencedDeclaration": 61,
                        "src": "777:3:0",
                        "typeDescriptions": {
                          "typeIdentifier": "t_magic_message",
                          "typeString": "msg"
                        }
                      },
                      "id": 10,
                      "isConstant": false,
                      "isLValue": false,
                      "isPure": false,
                      "lValueRequested": false,
                      "memberName": "sender",
                      "nodeType": "MemberAccess",
                      "referencedDeclaration": null,
                      "src": "777:10:0",
                      "typeDescriptions": {
                        "typeIdentifier": "t_address_payable",
                        "typeString": "address payable"
                      }
                    },
                    "src": "767:20:0",
                    "typeDescriptions": {
                      "typeIdentifier": "t_address",
                      "typeString": "address"
                    }
                  },
                  "id": 12,
                  "nodeType": "ExpressionStatement",
                  "src": "767:20:0"
                },
                {
                  "expression": {
                    "argumentTypes": null,
                    "id": 15,
                    "isConstant": false,
                    "isLValue": false,
                    "isPure": false,
                    "lValueRequested": false,
                    "leftHandSide": {
                      "argumentTypes": null,
                      "id": 13,
                      "name": "greeting",
                      "nodeType": "Identifier",
                      "overloadedDeclarations": [],
                      "referencedDeclaration": 5,
                      "src": "797:8:0",
                      "typeDescriptions": {
                        "typeIdentifier": "t_string_storage",
                        "typeString": "string storage ref"
                      }
                    },
                    "nodeType": "Assignment",
                    "operator": "=",
                    "rightHandSide": {
                      "argumentTypes": null,
                      "hexValue": "68656c6c6f20776f726c64",
                      "id": 14,
                      "isConstant": false,
                      "isLValue": false,
                      "isPure": true,
                      "kind": "string",
                      "lValueRequested": false,
                      "nodeType": "Literal",
                      "src": "808:13:0",
                      "subdenomination": null,
                      "typeDescriptions": {
                        "typeIdentifier": "t_stringliteral_47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad",
                        "typeString": "literal_string \"hello world\""
                      },
                      "value": "hello world"
                    },
                    "src": "797:24:0",
                    "typeDescriptions": {
                      "typeIdentifier": "t_string_storage",
                      "typeString": "string storage ref"
                    }
                  },
                  "id": 16,
                  "nodeType": "ExpressionStatement",
                  "src": "797:24:0"
                }
              ]
            },
            "documentation": null,
            "id": 18,
            "implemented": true,
            "kind": "function",
            "modifiers": [],
            "name": "constuctor",
            "nodeType": "FunctionDefinition",
            "parameters": {
              "id": 6,
              "nodeType": "ParameterList",
              "parameters": [],
              "src": "747:2:0"
            },
            "returnParameters": {
              "id": 7,
              "nodeType": "ParameterList",
              "parameters": [],
              "src": "757:0:0"
            },
            "scope": 46,
            "src": "728:100:0",
            "stateMutability": "nonpayable",
            "superFunction": null,
            "visibility": "public"
          },
          {
            "body": {
              "id": 25,
              "nodeType": "Block",
              "src": "887:37:0",
              "statements": [
                {
                  "expression": {
                    "argumentTypes": null,
                    "hexValue": "68656c6c6f20776f726c64",
                    "id": 23,
                    "isConstant": false,
                    "isLValue": false,
                    "isPure": true,
                    "kind": "string",
                    "lValueRequested": false,
                    "nodeType": "Literal",
                    "src": "904:13:0",
                    "subdenomination": null,
                    "typeDescriptions": {
                      "typeIdentifier": "t_stringliteral_47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad",
                      "typeString": "literal_string \"hello world\""
                    },
                    "value": "hello world"
                  },
                  "functionReturnParameters": 22,
                  "id": 24,
                  "nodeType": "Return",
                  "src": "897:20:0"
                }
              ]
            },
            "documentation": null,
            "id": 26,
            "implemented": true,
            "kind": "function",
            "modifiers": [],
            "name": "greet",
            "nodeType": "FunctionDefinition",
            "parameters": {
              "id": 19,
              "nodeType": "ParameterList",
              "parameters": [],
              "src": "848:2:0"
            },
            "returnParameters": {
              "id": 22,
              "nodeType": "ParameterList",
              "parameters": [
                {
                  "constant": false,
                  "id": 21,
                  "name": "",
                  "nodeType": "VariableDeclaration",
                  "scope": 26,
                  "src": "872:13:0",
                  "stateVariable": false,
                  "storageLocation": "memory",
                  "typeDescriptions": {
                    "typeIdentifier": "t_string_memory_ptr",
                    "typeString": "string"
                  },
                  "typeName": {
                    "id": 20,
                    "name": "string",
                    "nodeType": "ElementaryTypeName",
                    "src": "872:6:0",
                    "typeDescriptions": {
                      "typeIdentifier": "t_string_storage_ptr",
                      "typeString": "string"
                    }
                  },
                  "value": null,
                  "visibility": "internal"
                }
              ],
              "src": "871:15:0"
            },
            "scope": 46,
            "src": "834:90:0",
            "stateMutability": "view",
            "superFunction": null,
            "visibility": "public"
          },
          {
            "body": {
              "id": 34,
              "nodeType": "Block",
              "src": "1105:36:0",
              "statements": [
                {
                  "expression": {
                    "argumentTypes": null,
                    "expression": {
                      "argumentTypes": null,
                      "id": 31,
                      "name": "block",
                      "nodeType": "Identifier",
                      "overloadedDeclarations": [],
                      "referencedDeclaration": 51,
                      "src": "1122:5:0",
                      "typeDescriptions": {
                        "typeIdentifier": "t_magic_block",
                        "typeString": "block"
                      }
                    },
                    "id": 32,
                    "isConstant": false,
                    "isLValue": false,
                    "isPure": false,
                    "lValueRequested": false,
                    "memberName": "number",
                    "nodeType": "MemberAccess",
                    "referencedDeclaration": null,
                    "src": "1122:12:0",
                    "typeDescriptions": {
                      "typeIdentifier": "t_uint256",
                      "typeString": "uint256"
                    }
                  },
                  "functionReturnParameters": 30,
                  "id": 33,
                  "nodeType": "Return",
                  "src": "1115:19:0"
                }
              ]
            },
            "documentation": null,
            "id": 35,
            "implemented": true,
            "kind": "function",
            "modifiers": [],
            "name": "getBlockNumber",
            "nodeType": "FunctionDefinition",
            "parameters": {
              "id": 27,
              "nodeType": "ParameterList",
              "parameters": [],
              "src": "1075:2:0"
            },
            "returnParameters": {
              "id": 30,
              "nodeType": "ParameterList",
              "parameters": [
                {
                  "constant": false,
                  "id": 29,
                  "name": "",
                  "nodeType": "VariableDeclaration",
                  "scope": 35,
                  "src": "1099:4:0",
                  "stateVariable": false,
                  "storageLocation": "default",
                  "typeDescriptions": {
                    "typeIdentifier": "t_uint256",
                    "typeString": "uint256"
                  },
                  "typeName": {
                    "id": 28,
                    "name": "uint",
                    "nodeType": "ElementaryTypeName",
                    "src": "1099:4:0",
                    "typeDescriptions": {
                      "typeIdentifier": "t_uint256",
                      "typeString": "uint256"
                    }
                  },
                  "value": null,
                  "visibility": "internal"
                }
              ],
              "src": "1098:6:0"
            },
            "scope": 46,
            "src": "1052:89:0",
            "stateMutability": "view",
            "superFunction": null,
            "visibility": "public"
          },
          {
            "body": {
              "id": 44,
              "nodeType": "Block",
              "src": "1203:40:0",
              "statements": [
                {
                  "expression": {
                    "argumentTypes": null,
                    "id": 42,
                    "isConstant": false,
                    "isLValue": false,
                    "isPure": false,
                    "lValueRequested": false,
                    "leftHandSide": {
                      "argumentTypes": null,
                      "id": 40,
                      "name": "greeting",
                      "nodeType": "Identifier",
                      "overloadedDeclarations": [],
                      "referencedDeclaration": 5,
                      "src": "1213:8:0",
                      "typeDescriptions": {
                        "typeIdentifier": "t_string_storage",
                        "typeString": "string storage ref"
                      }
                    },
                    "nodeType": "Assignment",
                    "operator": "=",
                    "rightHandSide": {
                      "argumentTypes": null,
                      "id": 41,
                      "name": "_newgreeting",
                      "nodeType": "Identifier",
                      "overloadedDeclarations": [],
                      "referencedDeclaration": 37,
                      "src": "1224:12:0",
                      "typeDescriptions": {
                        "typeIdentifier": "t_string_memory_ptr",
                        "typeString": "string memory"
                      }
                    },
                    "src": "1213:23:0",
                    "typeDescriptions": {
                      "typeIdentifier": "t_string_storage",
                      "typeString": "string storage ref"
                    }
                  },
                  "id": 43,
                  "nodeType": "ExpressionStatement",
                  "src": "1213:23:0"
                }
              ]
            },
            "documentation": null,
            "id": 45,
            "implemented": true,
            "kind": "function",
            "modifiers": [],
            "name": "setGreeting",
            "nodeType": "FunctionDefinition",
            "parameters": {
              "id": 38,
              "nodeType": "ParameterList",
              "parameters": [
                {
                  "constant": false,
                  "id": 37,
                  "name": "_newgreeting",
                  "nodeType": "VariableDeclaration",
                  "scope": 45,
                  "src": "1168:26:0",
                  "stateVariable": false,
                  "storageLocation": "memory",
                  "typeDescriptions": {
                    "typeIdentifier": "t_string_memory_ptr",
                    "typeString": "string"
                  },
                  "typeName": {
                    "id": 36,
                    "name": "string",
                    "nodeType": "ElementaryTypeName",
                    "src": "1168:6:0",
                    "typeDescriptions": {
                      "typeIdentifier": "t_string_storage_ptr",
                      "typeString": "string"
                    }
                  },
                  "value": null,
                  "visibility": "internal"
                }
              ],
              "src": "1167:28:0"
            },
            "returnParameters": {
              "id": 39,
              "nodeType": "ParameterList",
              "parameters": [],
              "src": "1203:0:0"
            },
            "scope": 46,
            "src": "1147:96:0",
            "stateMutability": "nonpayable",
            "superFunction": null,
            "visibility": "public"
          }
        ],
        "scope": 47,
        "src": "334:911:0"
      }
    ],
    "src": "309:937:0"
  },
  "legacyAST": {
    "attributes": {
      "absolutePath": "/home/eoin/projects/provide/truffle/contracts/greeter.sol",
      "exportedSymbols": {
        "Greeter": [
          46
        ]
      }
    },
    "children": [
      {
        "attributes": {
          "literals": [
            "solidity",
            "^",
            "0.5",
            ".16"
          ]
        },
        "id": 1,
        "name": "PragmaDirective",
        "src": "309:24:0"
      },
      {
        "attributes": {
          "baseContracts": [
            null
          ],
          "contractDependencies": [
            null
          ],
          "contractKind": "contract",
          "documentation": null,
          "fullyImplemented": true,
          "linearizedBaseContracts": [
            46
          ],
          "name": "Greeter",
          "scope": 47
        },
        "children": [
          {
            "attributes": {
              "constant": false,
              "name": "creator",
              "scope": 46,
              "stateVariable": true,
              "storageLocation": "default",
              "type": "address",
              "value": null,
              "visibility": "internal"
            },
            "children": [
              {
                "attributes": {
                  "name": "address",
                  "stateMutability": "nonpayable",
                  "type": "address"
                },
                "id": 2,
                "name": "ElementaryTypeName",
                "src": "463:7:0"
              }
            ],
            "id": 3,
            "name": "VariableDeclaration",
            "src": "463:15:0"
          },
          {
            "attributes": {
              "constant": false,
              "name": "greeting",
              "scope": 46,
              "stateVariable": true,
              "storageLocation": "default",
              "type": "string",
              "value": null,
              "visibility": "internal"
            },
            "children": [
              {
                "attributes": {
                  "name": "string",
                  "type": "string"
                },
                "id": 4,
                "name": "ElementaryTypeName",
                "src": "604:6:0"
              }
            ],
            "id": 5,
            "name": "VariableDeclaration",
            "src": "604:15:0"
          },
          {
            "attributes": {
              "documentation": null,
              "implemented": true,
              "isConstructor": false,
              "kind": "function",
              "modifiers": [
                null
              ],
              "name": "constuctor",
              "scope": 46,
              "stateMutability": "nonpayable",
              "superFunction": null,
              "visibility": "public"
            },
            "children": [
              {
                "attributes": {
                  "parameters": [
                    null
                  ]
                },
                "children": [],
                "id": 6,
                "name": "ParameterList",
                "src": "747:2:0"
              },
              {
                "attributes": {
                  "parameters": [
                    null
                  ]
                },
                "children": [],
                "id": 7,
                "name": "ParameterList",
                "src": "757:0:0"
              },
              {
                "children": [
                  {
                    "children": [
                      {
                        "attributes": {
                          "argumentTypes": null,
                          "isConstant": false,
                          "isLValue": false,
                          "isPure": false,
                          "lValueRequested": false,
                          "operator": "=",
                          "type": "address"
                        },
                        "children": [
                          {
                            "attributes": {
                              "argumentTypes": null,
                              "overloadedDeclarations": [
                                null
                              ],
                              "referencedDeclaration": 3,
                              "type": "address",
                              "value": "creator"
                            },
                            "id": 8,
                            "name": "Identifier",
                            "src": "767:7:0"
                          },
                          {
                            "attributes": {
                              "argumentTypes": null,
                              "isConstant": false,
                              "isLValue": false,
                              "isPure": false,
                              "lValueRequested": false,
                              "member_name": "sender",
                              "referencedDeclaration": null,
                              "type": "address payable"
                            },
                            "children": [
                              {
                                "attributes": {
                                  "argumentTypes": null,
                                  "overloadedDeclarations": [
                                    null
                                  ],
                                  "referencedDeclaration": 61,
                                  "type": "msg",
                                  "value": "msg"
                                },
                                "id": 9,
                                "name": "Identifier",
                                "src": "777:3:0"
                              }
                            ],
                            "id": 10,
                            "name": "MemberAccess",
                            "src": "777:10:0"
                          }
                        ],
                        "id": 11,
                        "name": "Assignment",
                        "src": "767:20:0"
                      }
                    ],
                    "id": 12,
                    "name": "ExpressionStatement",
                    "src": "767:20:0"
                  },
                  {
                    "children": [
                      {
                        "attributes": {
                          "argumentTypes": null,
                          "isConstant": false,
                          "isLValue": false,
                          "isPure": false,
                          "lValueRequested": false,
                          "operator": "=",
                          "type": "string storage ref"
                        },
                        "children": [
                          {
                            "attributes": {
                              "argumentTypes": null,
                              "overloadedDeclarations": [
                                null
                              ],
                              "referencedDeclaration": 5,
                              "type": "string storage ref",
                              "value": "greeting"
                            },
                            "id": 13,
                            "name": "Identifier",
                            "src": "797:8:0"
                          },
                          {
                            "attributes": {
                              "argumentTypes": null,
                              "hexvalue": "68656c6c6f20776f726c64",
                              "isConstant": false,
                              "isLValue": false,
                              "isPure": true,
                              "lValueRequested": false,
                              "subdenomination": null,
                              "token": "string",
                              "type": "literal_string \"hello world\"",
                              "value": "hello world"
                            },
                            "id": 14,
                            "name": "Literal",
                            "src": "808:13:0"
                          }
                        ],
                        "id": 15,
                        "name": "Assignment",
                        "src": "797:24:0"
                      }
                    ],
                    "id": 16,
                    "name": "ExpressionStatement",
                    "src": "797:24:0"
                  }
                ],
                "id": 17,
                "name": "Block",
                "src": "757:71:0"
              }
            ],
            "id": 18,
            "name": "FunctionDefinition",
            "src": "728:100:0"
          },
          {
            "attributes": {
              "documentation": null,
              "implemented": true,
              "isConstructor": false,
              "kind": "function",
              "modifiers": [
                null
              ],
              "name": "greet",
              "scope": 46,
              "stateMutability": "view",
              "superFunction": null,
              "visibility": "public"
            },
            "children": [
              {
                "attributes": {
                  "parameters": [
                    null
                  ]
                },
                "children": [],
                "id": 19,
                "name": "ParameterList",
                "src": "848:2:0"
              },
              {
                "children": [
                  {
                    "attributes": {
                      "constant": false,
                      "name": "",
                      "scope": 26,
                      "stateVariable": false,
                      "storageLocation": "memory",
                      "type": "string",
                      "value": null,
                      "visibility": "internal"
                    },
                    "children": [
                      {
                        "attributes": {
                          "name": "string",
                          "type": "string"
                        },
                        "id": 20,
                        "name": "ElementaryTypeName",
                        "src": "872:6:0"
                      }
                    ],
                    "id": 21,
                    "name": "VariableDeclaration",
                    "src": "872:13:0"
                  }
                ],
                "id": 22,
                "name": "ParameterList",
                "src": "871:15:0"
              },
              {
                "children": [
                  {
                    "attributes": {
                      "functionReturnParameters": 22
                    },
                    "children": [
                      {
                        "attributes": {
                          "argumentTypes": null,
                          "hexvalue": "68656c6c6f20776f726c64",
                          "isConstant": false,
                          "isLValue": false,
                          "isPure": true,
                          "lValueRequested": false,
                          "subdenomination": null,
                          "token": "string",
                          "type": "literal_string \"hello world\"",
                          "value": "hello world"
                        },
                        "id": 23,
                        "name": "Literal",
                        "src": "904:13:0"
                      }
                    ],
                    "id": 24,
                    "name": "Return",
                    "src": "897:20:0"
                  }
                ],
                "id": 25,
                "name": "Block",
                "src": "887:37:0"
              }
            ],
            "id": 26,
            "name": "FunctionDefinition",
            "src": "834:90:0"
          },
          {
            "attributes": {
              "documentation": null,
              "implemented": true,
              "isConstructor": false,
              "kind": "function",
              "modifiers": [
                null
              ],
              "name": "getBlockNumber",
              "scope": 46,
              "stateMutability": "view",
              "superFunction": null,
              "visibility": "public"
            },
            "children": [
              {
                "attributes": {
                  "parameters": [
                    null
                  ]
                },
                "children": [],
                "id": 27,
                "name": "ParameterList",
                "src": "1075:2:0"
              },
              {
                "children": [
                  {
                    "attributes": {
                      "constant": false,
                      "name": "",
                      "scope": 35,
                      "stateVariable": false,
                      "storageLocation": "default",
                      "type": "uint256",
                      "value": null,
                      "visibility": "internal"
                    },
                    "children": [
                      {
                        "attributes": {
                          "name": "uint",
                          "type": "uint256"
                        },
                        "id": 28,
                        "name": "ElementaryTypeName",
                        "src": "1099:4:0"
                      }
                    ],
                    "id": 29,
                    "name": "VariableDeclaration",
                    "src": "1099:4:0"
                  }
                ],
                "id": 30,
                "name": "ParameterList",
                "src": "1098:6:0"
              },
              {
                "children": [
                  {
                    "attributes": {
                      "functionReturnParameters": 30
                    },
                    "children": [
                      {
                        "attributes": {
                          "argumentTypes": null,
                          "isConstant": false,
                          "isLValue": false,
                          "isPure": false,
                          "lValueRequested": false,
                          "member_name": "number",
                          "referencedDeclaration": null,
                          "type": "uint256"
                        },
                        "children": [
                          {
                            "attributes": {
                              "argumentTypes": null,
                              "overloadedDeclarations": [
                                null
                              ],
                              "referencedDeclaration": 51,
                              "type": "block",
                              "value": "block"
                            },
                            "id": 31,
                            "name": "Identifier",
                            "src": "1122:5:0"
                          }
                        ],
                        "id": 32,
                        "name": "MemberAccess",
                        "src": "1122:12:0"
                      }
                    ],
                    "id": 33,
                    "name": "Return",
                    "src": "1115:19:0"
                  }
                ],
                "id": 34,
                "name": "Block",
                "src": "1105:36:0"
              }
            ],
            "id": 35,
            "name": "FunctionDefinition",
            "src": "1052:89:0"
          },
          {
            "attributes": {
              "documentation": null,
              "implemented": true,
              "isConstructor": false,
              "kind": "function",
              "modifiers": [
                null
              ],
              "name": "setGreeting",
              "scope": 46,
              "stateMutability": "nonpayable",
              "superFunction": null,
              "visibility": "public"
            },
            "children": [
              {
                "children": [
                  {
                    "attributes": {
                      "constant": false,
                      "name": "_newgreeting",
                      "scope": 45,
                      "stateVariable": false,
                      "storageLocation": "memory",
                      "type": "string",
                      "value": null,
                      "visibility": "internal"
                    },
                    "children": [
                      {
                        "attributes": {
                          "name": "string",
                          "type": "string"
                        },
                        "id": 36,
                        "name": "ElementaryTypeName",
                        "src": "1168:6:0"
                      }
                    ],
                    "id": 37,
                    "name": "VariableDeclaration",
                    "src": "1168:26:0"
                  }
                ],
                "id": 38,
                "name": "ParameterList",
                "src": "1167:28:0"
              },
              {
                "attributes": {
                  "parameters": [
                    null
                  ]
                },
                "children": [],
                "id": 39,
                "name": "ParameterList",
                "src": "1203:0:0"
              },
              {
                "children": [
                  {
                    "children": [
                      {
                        "attributes": {
                          "argumentTypes": null,
                          "isConstant": false,
                          "isLValue": false,
                          "isPure": false,
                          "lValueRequested": false,
                          "operator": "=",
                          "type": "string storage ref"
                        },
                        "children": [
                          {
                            "attributes": {
                              "argumentTypes": null,
                              "overloadedDeclarations": [
                                null
                              ],
                              "referencedDeclaration": 5,
                              "type": "string storage ref",
                              "value": "greeting"
                            },
                            "id": 40,
                            "name": "Identifier",
                            "src": "1213:8:0"
                          },
                          {
                            "attributes": {
                              "argumentTypes": null,
                              "overloadedDeclarations": [
                                null
                              ],
                              "referencedDeclaration": 37,
                              "type": "string memory",
                              "value": "_newgreeting"
                            },
                            "id": 41,
                            "name": "Identifier",
                            "src": "1224:12:0"
                          }
                        ],
                        "id": 42,
                        "name": "Assignment",
                        "src": "1213:23:0"
                      }
                    ],
                    "id": 43,
                    "name": "ExpressionStatement",
                    "src": "1213:23:0"
                  }
                ],
                "id": 44,
                "name": "Block",
                "src": "1203:40:0"
              }
            ],
            "id": 45,
            "name": "FunctionDefinition",
            "src": "1147:96:0"
          }
        ],
        "id": 46,
        "name": "ContractDefinition",
        "src": "334:911:0"
      }
    ],
    "id": 47,
    "name": "SourceUnit",
    "src": "309:937:0"
  },
  "compiler": {
    "name": "solc",
    "version": "0.5.16+commit.9c3226ce.Emscripten.clang"
  },
  "networks": {},
  "schemaVersion": "3.3.4",
  "updatedAt": "2021-02-23T17:11:35.424Z",
  "devdoc": {
    "methods": {}
  },
  "userdoc": {
    "methods": {}
  }
}`

const shuttleABIArtifact = `[
	{
		"constant": true,
		"inputs": [
			{
				"name": "interfaceId",
				"type": "bytes4"
			}
		],
		"name": "supportsInterface",
		"outputs": [
			{
				"name": "",
				"type": "bool"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "_address",
				"type": "address"
			}
		],
		"name": "getOrg",
		"outputs": [
			{
				"name": "",
				"type": "address"
			},
			{
				"name": "",
				"type": "bytes32"
			},
			{
				"name": "",
				"type": "bytes"
			},
			{
				"name": "",
				"type": "bytes"
			},
			{
				"name": "",
				"type": "bytes"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "_address",
				"type": "address"
			},
			{
				"name": "_name",
				"type": "bytes32"
			},
			{
				"name": "_messengerEndpoint",
				"type": "bytes"
			},
			{
				"name": "_whisperKey",
				"type": "bytes"
			},
			{
				"name": "_zkpPublicKey",
				"type": "bytes"
			}
		],
		"name": "registerOrg",
		"outputs": [
			{
				"name": "",
				"type": "bool"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": null,
		"name": "getOrgCount",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "interfaceHash",
				"type": "bytes32"
			},
			{
				"name": "addr",
				"type": "address"
			}
		],
		"name": "canImplementInterfaceForAddress",
		"outputs": [
			{
				"name": "",
				"type": "bytes32"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "_newManager",
				"type": "address"
			}
		],
		"name": "assignManager",
		"outputs": null,
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "addr",
				"type": "address"
			},
			{
				"name": "_interfaceLabel",
				"type": "string"
			}
		],
		"name": "interfaceAddr",
		"outputs": [
			{
				"name": "",
				"type": "address"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "_groupName",
				"type": "bytes32"
			},
			{
				"name": "_tokenAddress",
				"type": "address"
			},
			{
				"name": "_shieldAddress",
				"type": "address"
			},
			{
				"name": "_verifierAddress",
				"type": "address"
			}
		],
		"name": "registerInterfaces",
		"outputs": [
			{
				"name": "",
				"type": "bool"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": null,
		"name": "owner",
		"outputs": [
			{
				"name": "",
				"type": "address"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": null,
		"name": "_owner",
		"outputs": [
			{
				"name": "",
				"type": "address"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": null,
		"name": "getManager",
		"outputs": [
			{
				"name": "",
				"type": "address"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": null,
		"name": "getInterfaceAddresses",
		"outputs": [
			{
				"name": "",
				"type": "bytes32[]"
			},
			{
				"name": "",
				"type": "address[]"
			},
			{
				"name": "",
				"type": "address[]"
			},
			{
				"name": "",
				"type": "address[]"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": null,
		"name": "getInterfaces",
		"outputs": [
			{
				"name": "",
				"type": "bytes4"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "newOwner",
				"type": "address"
			}
		],
		"name": "transferOwnership",
		"outputs": null,
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": null,
		"name": "setInterfaces",
		"outputs": [
			{
				"name": "",
				"type": "bool"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"name": "_erc1820",
				"type": "address"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "constructor"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"name": "_name",
				"type": "bytes32"
			},
			{
				"indexed": false,
				"name": "_address",
				"type": "address"
			},
			{
				"indexed": false,
				"name": "_messengerEndpoint",
				"type": "bytes"
			},
			{
				"indexed": false,
				"name": "_whisperKey",
				"type": "bytes"
			},
			{
				"indexed": false,
				"name": "_zkpPublicKey",
				"type": "bytes"
			}
		],
		"name": "RegisterOrg",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"name": "previousOwner",
				"type": "address"
			},
			{
				"indexed": false,
				"name": "newOwner",
				"type": "address"
			}
		],
		"name": "OwnershipTransferred",
		"type": "event"
	}
]`
