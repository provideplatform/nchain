package contract

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/provideapp/goldmine/common"
	provide "github.com/provideservices/provide-go"
)

const swarmHashPrefix string = "a165627a7a72305820" // 0xa1 0x65 'b' 'z' 'z' 'r' '0' 0x58 0x20

func shellOut(bash string) ([]byte, error) {
	cmd := exec.Command("bash", "-c", bash)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Output()
}

func getContractABI(compiledContract map[string]interface{}) ([]interface{}, error) {
	abiJSON, ok := compiledContract["abi"].(string)
	if !ok {
		return nil, fmt.Errorf("Failed to retrieve contract ABI from compiled contract")
	}

	_abi := []interface{}{}
	err := json.Unmarshal([]byte(abiJSON), &_abi)
	return _abi, err
}

func getContractAssembly(compiledContract map[string]interface{}) (map[string]interface{}, error) {
	contractAsm, ok := compiledContract["asm"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Unable to read assembly from compiled contract: %s", compiledContract)
	}
	return contractAsm, nil
}

func getContractOpcodes(compiledContract map[string]interface{}) (string, error) {
	opcodes, ok := compiledContract["opcodes"].(string)
	if !ok || opcodes == "" {
		return "", fmt.Errorf("Unable to read opcodes from compiled contract: %s", compiledContract)
	}
	return opcodes, nil
}

func getContractFingerprint(compiledContract map[string]interface{}) (*string, error) {
	bytecode, err := getContractBytecode(compiledContract)
	if err != nil {
		return nil, fmt.Errorf("Unable to read contract bytecode; %s", err.Error())
	}
	fingerprintIdx := strings.LastIndex(string(bytecode), swarmHashPrefix)
	if fingerprintIdx == -1 {
		return nil, fmt.Errorf("Unable to resolve contract swarm hash for compiled contract: %s", compiledContract)
	}
	fingerprint := string(bytecode)[fingerprintIdx+len(swarmHashPrefix):]
	fingerprint = fingerprint[0 : len(fingerprint)-4]
	return &fingerprint, nil
}

func getContractSource(flattenedSrc, compilerSemanticVersion string, compiledContract map[string]interface{}, contract string) (*string, error) {
	src := fmt.Sprintf("pragma solidity ^%s\n\n", compilerSemanticVersion)
	srcmap, err := getContractSourcemap(compiledContract)
	if err != nil {
		return nil, fmt.Errorf("Unable to read contract sourcemap; %s", err.Error())
	}
	if *srcmap == "" {
		return nil, fmt.Errorf("Contract sourcemap was empty: %s", contract)
	}
	mapParts := strings.Split(*srcmap, ":")
	begin, _ := strconv.Atoi(mapParts[0])
	end, _ := strconv.Atoi(mapParts[1])
	end = begin + end
	src = fmt.Sprintf("%s%s", src, flattenedSrc[begin:end])
	return &src, nil
}

func getContractSourcemap(compiledContract map[string]interface{}) (*string, error) {
	srcmap, ok := compiledContract["srcmap"].(string)
	if !ok || srcmap == "" {
		return nil, fmt.Errorf("Unable to read contract sourcemap from compiled contract: %s", compiledContract)
	}
	return &srcmap, nil
}

func getContractSourceMeta(compilerOutput map[string]interface{}, contract string) (map[string]interface{}, error) {
	contractSources, ok := compilerOutput["sources"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Unable to read contract sources from compiled contract: %s", compilerOutput)
	}
	contractSource, ok := contractSources[contract].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Unable to read contract source for contract: %s", contract)
	}
	return contractSource, nil
}

func getContractDependencies(src, compilerVersion string, compilerOutput map[string]interface{}, contract string) (map[string]interface{}, error) {
	source, err := getContractSourceMeta(compilerOutput, "<stdin>")
	if err != nil {
		common.Log.Warningf("Failed to retrieve contract sources from compiled contract")
		return nil, err
	}
	ast, ok := source["AST"].(map[string]interface{})

	astExports, ok := ast["exportedSymbols"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Failed to retrieve contract exports from compiled contract AST")
	}

	reentrant := false
	if resolvedExports, ok := astExports[contract].([]interface{}); ok {
		reentrant = len(resolvedExports) > 1
	}

	exports := map[int]string{}
	for name, ids := range astExports {
		if strings.Contains(contract, name) {
			continue
		}

		exportIds := make([]int64, 0)
		for i := range ids.([]interface{}) {
			exportIds = append(exportIds, int64(ids.([]interface{})[i].(float64)))
		}
		exports[int(exportIds[0])] = name
	}

	nodes, ok := ast["nodes"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("Failed to retrieve contract nodes from compiled contract AST")
	}
	if len(nodes) <= 1 {
		return nil, fmt.Errorf("Failed to retrieve contract dependencies from compiled contract nodes; malformed AST?")
	}

	dependencies := map[string]interface{}{}

	for i := range exports {
		dependencyContractKey := exports[i]
		dependencyContractKeyParts := strings.Split(dependencyContractKey, ":")
		dependencyContractName := dependencyContractKeyParts[len(dependencyContractKeyParts)-1]

		_dependencyContractKey := fmt.Sprintf("<stdin>:%s", dependencyContractKey)

		dependencyContract := compilerOutput["contracts"].(map[string]interface{})[_dependencyContractKey].(map[string]interface{})
		dependencyContractABI, _ := getContractABI(dependencyContract)
		dependencyContractBytecode, _ := getContractBytecode(dependencyContract)
		dependencyContractAssembly, _ := getContractAssembly(dependencyContract)
		dependencyContractOpcodes, _ := getContractOpcodes(dependencyContract)
		dependencyContractRaw, _ := json.Marshal(dependencyContract)
		dependencyContractSource, _ := getContractSource(src, compilerVersion, dependencyContract, dependencyContractName)
		dependencyContractFingerprint, _ := getContractFingerprint(dependencyContract)

		var deps map[string]interface{}

		if reentrant {
			deps, _ = getContractDependencies(src, compilerVersion, compilerOutput, dependencyContractName)
		}

		dependencies[dependencyContractName] = &provide.CompiledArtifact{
			Name:        dependencyContractName,
			ABI:         dependencyContractABI,
			Assembly:    dependencyContractAssembly,
			Bytecode:    string(dependencyContractBytecode),
			Deps:        deps,
			Opcodes:     dependencyContractOpcodes,
			Raw:         json.RawMessage(dependencyContractRaw),
			Source:      dependencyContractSource,
			Fingerprint: dependencyContractFingerprint,
		}
	}

	return dependencies, nil
}

func getContractBytecode(compiledContract map[string]interface{}) ([]byte, error) {
	bytecode, ok := compiledContract["bin"].(string)
	if !ok {
		return nil, fmt.Errorf("Unable to read bytecode from compiled contract: %s", compiledContract)
	}
	return []byte(bytecode), nil
}

func parseContractABI(contractABIJSON []byte) (*abi.ABI, error) {
	abival, err := abi.JSON(strings.NewReader(string(contractABIJSON)))
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize ABI from contract params to json; %s", err.Error())
	}

	return &abival, nil
}

func parseCompilerOutput(compilerOutputJSON []byte) (compiledContracts map[string]interface{}, err error) {
	combinedOutput := map[string]interface{}{}
	err = json.Unmarshal(compilerOutputJSON, &combinedOutput)
	return combinedOutput, err
}

func parseCompiledContracts(compilerOutputJSON []byte) (compiledContracts map[string]interface{}, err error) {
	combinedOutput, err := parseCompilerOutput(compilerOutputJSON)
	common.Log.Debugf("%s", combinedOutput)
	if err == nil && combinedOutput != nil {
		if compiledContracts, compiledContractsOk := combinedOutput["contracts"].(map[string]interface{}); compiledContractsOk {
			return compiledContracts, err
		}
	}
	return nil, err
}

func buildCompileCommand(source, compilerVersion string, optimizerRuns int) string {
	return fmt.Sprintf("echo -n \"$(cat <<-EOF\n%s\nEOF\n)\" | /usr/local/bin/solc-v%s --optimize --optimize-runs %d --pretty-json --metadata-literal --combined-json abi,asm,ast,bin,bin-runtime,compact-format,devdoc,hashes,interface,metadata,opcodes,srcmap,srcmap-runtime,userdoc -", source, compilerVersion, optimizerRuns)
	// TODO: run optimizer over certain sources if identified for frequent use via contract-internal CREATE opcodes
}

// compileContract compiles a smart contract or truffle project from source.
func compileSolidity(name, source string, constructorParams []interface{}, compilerOptimizerRuns int) (*provide.CompiledArtifact, error) {
	var err error

	// FIXME-- make wrapper around regexp
	// common.Log.Debugf("Compiled regex: %s", r)
	// groups := r.FindAllString(source, 1)
	// common.Log.Debugf("groups: %s", groups)
	// if len(groups) != 1 {
	// 	err = fmt.Errorf("Failed to find pragma directive for solidity contract compilation")
	// 	common.Log.Warning(err.Error())
	// 	return nil, err
	// }
	directiveParts := strings.Split(source, "pragma solidity ^")
	if len(directiveParts) != 2 {
		directiveParts = strings.Split(source, "pragma solidity ")
	}
	if len(directiveParts) != 2 {
		err = fmt.Errorf("Failed to find pragma directive for solidity contract compilation")
		common.Log.Warning(err.Error())
		return nil, err
	}

	compilerVersion := strings.Replace(directiveParts[1][0:len(directiveParts[1])-2], ";", "", 1)
	compilerVersion = strings.Split(compilerVersion, "\n")[0]
	common.Log.Debugf("Resolved compiler version: %s", compilerVersion)

	solcCmd := buildCompileCommand(source, compilerVersion, compilerOptimizerRuns)
	common.Log.Debugf("Built solc command: %s", solcCmd)

	stdOut, err := shellOut(solcCmd)
	if err != nil {
		return nil, fmt.Errorf("Failed to compile contract(s): %s; %s", name, err.Error())
	}

	common.Log.Debugf("Raw solc compiler output: %s", stdOut)

	compilerOutput, err := parseCompilerOutput(stdOut)
	contracts, err := parseCompiledContracts(stdOut)
	if err != nil {
		return nil, fmt.Errorf("Failed to compile contract(s): %s; %s", name, err.Error())
	}

	common.Log.Debugf("Compiled %d solidity contract(s) from source: %s", len(contracts), contracts)

	depGraph := map[string]interface{}{}
	var topLevelConstructor *abi.Method

	for key := range contracts {
		keyParts := strings.Split(key, ":")
		contractName := keyParts[len(keyParts)-1]
		contract := contracts[key].(map[string]interface{})

		parsedABI, _ := getContractABI(contract)
		_abi, _ := parseContractABI([]byte(contract["abi"].(string)))
		bytecode, _ := getContractBytecode(contract)
		assembly, _ := getContractAssembly(contract)
		opcodes, _ := getContractOpcodes(contract)
		raw, _ := json.Marshal(contract)
		src, _ := getContractSource(source, compilerVersion, contract, contractName)
		fingerprint, _ := getContractFingerprint(contract)

		contractDependencies, err := getContractDependencies(source, compilerVersion, compilerOutput, contractName)
		if err != nil {
			return nil, fmt.Errorf("WARNING: failed to retrieve contract dependencies for contract: %s", contractName)
		}

		depGraph[contractName] = &provide.CompiledArtifact{
			Name:        contractName,
			ABI:         parsedABI,
			Assembly:    assembly,
			Bytecode:    string(bytecode),
			Deps:        contractDependencies,
			Opcodes:     opcodes,
			Raw:         json.RawMessage(raw),
			Source:      src,
			Fingerprint: fingerprint,
		}

		if name == contractName {
			topLevelConstructor = &_abi.Constructor
		}
	}

	if topLevelConstructor == nil {
		return nil, fmt.Errorf("WARNING: no top-level contract resolved for %s", name)
	}

	var artifact *provide.CompiledArtifact // this is the artifact compatible with the provide-cli contract deployment and will be cached on disk temporarily

	var invocationSig string
	for depName, meta := range depGraph {
		if depName == name {
			bytecode := meta.(*provide.CompiledArtifact).Bytecode
			invocationSig = fmt.Sprintf("0x%s", string(bytecode))
			artifact = meta.(*provide.CompiledArtifact)
		}
	}

	argvLength := topLevelConstructor.Inputs.LengthNonIndexed()
	if len(constructorParams) != argvLength {
		return nil, fmt.Errorf("Constructor for %s contract requires %d parameters at compile-time; given: %d", name, argvLength, len(constructorParams))
	}

	encodedArgv, err := provide.EVMEncodeABI(topLevelConstructor, constructorParams...)
	if err != nil {
		return nil, fmt.Errorf("WARNING: failed to encode %d parameters prior to compiling contract: %s; %s", len(constructorParams), name, err.Error())
	}

	invocationSig = fmt.Sprintf("%s%s", invocationSig, ethcommon.ToHex(encodedArgv)[8:])
	artifact.Bytecode = invocationSig

	return artifact, nil
}
