package contract_api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"reflect"
	"repo.hovitos.engineering/MTN/go-solidity/utility"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SolidityContract struct {
	baseBody              map[string]string
	name                  string
	from                  string
	rpcURL                string
	compiledContract      *RpcCompiledContract
	contractAddress       string
	filter_id             string
	noEventlistener       bool
	tx_delay_toleration   int
	sync_delay_toleration int
	integration_test      int
	logger                *utility.DebugTrace
	logBlockchainStats    string
}


// === global state used to detect when we havent seen a block in a while ===
type blockSync struct {
	lastBlockTime  int64	// The unix time in seconds when blockNumber was last updated
	blockNumber    string 	// The last block that was seen
	blockStable    string   // The last block taht can be read from
}

var global_block_state_lock sync.Mutex
var global_block_state blockSync
var no_recent_blocks int
var block_read_delay int
var block_update_delay int

func update_block(blockNumber string) {
	global_block_state_lock.Lock()
	defer global_block_state_lock.Unlock()

	if global_block_state.blockNumber == "" || blockNumber != global_block_state.blockNumber {
		global_block_state.lastBlockTime = time.Now().Unix()
		global_block_state.blockNumber = blockNumber
		block,_ := strconv.ParseUint(blockNumber[2:], 16, 32)
		block = block - uint64(block_read_delay)
		global_block_state.blockStable = fmt.Sprintf("0x%x", block)
	}
}

func blocks_stopped() bool {
	last_block_time := global_block_state.lastBlockTime
	delta := time.Now().Unix()-last_block_time
	if int(delta) >= no_recent_blocks {
		return true
	}
	return false
}

func SolidityContractFactory(name string) *SolidityContract {
	sc := new(SolidityContract)
	sc.name = name
	sc.rpcURL = "http://localhost:8545"
	sc.compiledContract = nil
	sc.baseBody = make(map[string]string)
	sc.baseBody["jsonrpc"] = "2.0"
	sc.baseBody["id"] = "1"
	sc.logger = utility.DebugTraceFactory(os.Getenv("mtn_soliditycontract"), "")
	var delay int
	var err error
	if delay, err = strconv.Atoi(os.Getenv("mtn_soliditycontract_txdelay")); err != nil || delay == 0 {
		delay = 180
	}
	sc.tx_delay_toleration = delay
	var sync_delay int
	if sync_delay, err = strconv.Atoi(os.Getenv("mtn_soliditycontract_syncdelay")); err != nil || sync_delay == 0 {
		sync_delay = 180
	}
	sc.sync_delay_toleration = sync_delay
	var integration_test int
	if integration_test, err = strconv.Atoi(os.Getenv("mtn_soliditycontract_integration")); err != nil || integration_test == 0 {
		integration_test = 0
	}
	sc.integration_test = integration_test
	
	if no_recent_blocks, err = strconv.Atoi(os.Getenv("mtn_soliditycontract_no_recent_blocks")); err != nil || no_recent_blocks == 0 {
		no_recent_blocks = 300
	}

	sc.logBlockchainStats = os.Getenv("mtn_soliditycontract_logstats")

	if block_read_delay, err = strconv.Atoi(os.Getenv("mtn_soliditycontract_block_read_delay")); err != nil {
		block_read_delay = 3
	}

	if block_update_delay, err = strconv.Atoi(os.Getenv("mtn_soliditycontract_block_update_delay")); err != nil {
		block_update_delay = 10
	}

	return sc
}

func (self *SolidityContract) get_stable_block() string {
	delta := time.Now().Unix()-global_block_state.lastBlockTime
	if int(delta) >= block_update_delay {
		if _, err := self.get_current_block(); err != nil {
			self.logger.Debug("Debug", err)
		}
	}

	return global_block_state.blockStable
}

func (self *SolidityContract) dump_block_info() {
	self.logger.Debug("Debug", fmt.Sprintf("Current block %v, stable block %v", global_block_state.blockNumber, global_block_state.blockStable))
}

func (self *SolidityContract) Deploy_contract(from string, block_chain_url string) (bool, error) {
	self.logger.Debug("Entry", from, block_chain_url)
	result, contract_string, tx_address, err := false, "", "", error(nil)

	if from == "" {
		err = &DeployError{fmt.Sprintf("Must specify ethereum account address as first parameter, specified %v.", from)}
	} else {
		self.from = from
		if block_chain_url != "" {
			self.rpcURL = block_chain_url
		}
		if self.compiledContract == nil {
			if contract_string, err = self.get_contract_as_string(); err == nil {
				self.compiledContract, err = self.compile_contract(contract_string)
			}
		}

		if err == nil {
			if tx_address, err = self.create_contract(); err == nil {
				if self.contractAddress, err = self.get_contract(tx_address); err == nil {
					if self.filter_id, err = self.establish_event_listener(); err == nil {
						result = true
					}
				}
			}
		}

	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", result)
	return result, err
}

func (self *SolidityContract) Load_contract(from string, block_chain_url string) (bool, error) {
	self.logger.Debug("Entry", from, block_chain_url)
	result, contract_string, err := false, "", error(nil)

	if from == "" {
		err = &LoadError{fmt.Sprintf("Must specify ethereum account address as first parameter, specified %v.", from)}
	} else {
		self.from = from
		if block_chain_url != "" {
			self.rpcURL = block_chain_url
		}
		if self.compiledContract == nil {
			if contract_string, err = self.get_contract_as_string(); err == nil {
				if self.compiledContract, err = self.compile_contract(contract_string); err == nil {
					result = true
				}
			}
		} else {
			result = true
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", result)
	return result, err
}

func (self *SolidityContract) Invoke_method(method_name string, params []interface{}) (interface{}, error) {
	self.logger.Debug("Entry", method_name, params)
	out, method_id, invocation_string, eth_method, hex_sig, found, err := "", "", "", "", "", false, error(nil)
	var result interface{}
	var rpcResp *rpcResponse = new(rpcResponse)

	if (self.contractAddress == "") {
		err = &RPCError{fmt.Sprintf("This object has no contract address. Please use Set_contract_address() before invoking any contract methods.\n")}
	} else if (self.compiledContract == nil ) {
		err = &RPCError{fmt.Sprintf("This object has no compiled contract. Please use Load_contract() before invoking any contract methods.\n")}
	}

	if err == nil {
		if hex_sig, err = self.get_method_sig(method_name); err == nil {
			if out, err = self.Call_rpc_api("web3_sha3", hex_sig); err == nil {
				if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
					if rpcResp.Error.Message != "" {
						err = &RPCError{fmt.Sprintf("RPC hash of method signature for %v failed, error: %v.", method_name, rpcResp.Error.Message)}
					} else {
						method_id = rpcResp.Result.(string)[:10]
					}
				}
			}
		}
	}

	if err == nil {
		if invocation_string, err = self.encodeInputString(method_name, params); err == nil {
			invocation_string = method_id + invocation_string
			eth_method = "eth_call"
			if !self.is_constant(method_name) {
				eth_method = "eth_sendTransaction"
			}
		}
	}

	// Let's make sure our ethereum instance is still working correctly
	if eth_method == "eth_sendTransaction" {
		err = self.check_eth_status()
	}

	if err == nil {

		p := make(map[string]string)
		p["from"] = self.from
		p["to"] = self.contractAddress
		p["gas"] = "0x16e360"
		p["data"] = invocation_string

		if out, err = self.Call_rpc_api(eth_method, p); err == nil {
			if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
				if rpcResp.Error.Message != "" {
					err = &RPCError{fmt.Sprintf("RPC invocation of %v failed, error: %v.", method_name, rpcResp.Error.Message)}
				} else {
					if !self.is_constant(method_name) {
						tx_address := rpcResp.Result.(string)
						var rpcTResp *rpcGetTransactionResponse = new(rpcGetTransactionResponse)

						start_timer := time.Now()
						for !found && err == nil {
							if out, err = self.Call_rpc_api("eth_getTransactionReceipt", tx_address); err == nil {
								if err = json.Unmarshal([]byte(out), rpcTResp); err == nil {
									if rpcTResp.Error.Message != "" {
										err = &RPCError{fmt.Sprintf("RPC transaction receipt for tx %v, invoking %v returned an error: %v.", tx_address, method_name, rpcResp.Error.Message)}
									} else {
										//self.logger.Debug("Debug",rpcTResp.Result)
										if rpcTResp.Result.BlockNumber != "" {
											result = 0
											found = true
											update_block(rpcTResp.Result.BlockNumber)
											self.log_stats(rpcTResp)
										} else {
											delta := time.Now().Sub(start_timer).Seconds()
											if int(delta) < self.tx_delay_toleration {
												self.logger.Debug("Debug", fmt.Sprintf("Waiting for transaction %v to run for %v seconds.", tx_address, delta))
												time.Sleep(5000 * time.Millisecond)
												err = self.check_eth_status()
											} else {
												err = &RPCError{fmt.Sprintf("RPC transaction receipt timed out for tx %v, invoking %v after %v seconds.", tx_address, method_name, delta)}
											}
										}
									}
								}
							}
						}
					} else {
						if rpcResp.Result != "0x" {
							result, err = self.decodeOutputString(method_name, rpcResp.Result.(string)[2:])
						} else {
							err = &RPCError{fmt.Sprintf("RPC invocation eth_call returned %v, the EVM probably failed executing method %v.", rpcResp.Result, method_name)}
						}
					}
				}
			}
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", result)
	return result, err
}

func (self *SolidityContract) Wait_for_event(event_code []uint64, related_contract string) ([]uint64, error) {
	self.logger.Debug("Entry", "")
	out, found, err := "", false, error(nil)
	var rpcResp *rpcGetFilterChangesResponse = new(rpcGetFilterChangesResponse)
	var ev_code uint64
	var ret_ev_code []uint64
	ret_ev_code = make([]uint64, 0, 10)

	for !found && err == nil {
		if out, err = self.Call_rpc_api("eth_getFilterChanges", self.filter_id); err == nil {
			if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
				if rpcResp.Error.Message != "" {
					err = &RPCError{fmt.Sprintf("RPC receive events for %v failed, error: %v.", self.contractAddress, rpcResp.Error.Message)}
				} else {
					//self.logger.Debug("Debug",rpcResp.Result)
					if len(rpcResp.Result) > 0 {
						for _, ev := range rpcResp.Result {
							if len(ev.Topics) > 2 {
								if ev_code, err = strconv.ParseUint(ev.Topics[1][2:], 16, 32); err != nil {
									err = &RPCError{fmt.Sprintf("RPC event code not parse-able %v, error: %v.", ev.Topics[1], err)}
									break
								} else {
									for _, requested_evc := range event_code {
										if ev_code == requested_evc && ev.Topics[2][26:] == related_contract[2:] {
											found = true
											ret_ev_code = append(ret_ev_code, ev_code)
											break
										}
									}

								}
							}
						}
					}
					if !found {
						self.logger.Debug("Debug", fmt.Sprintf("Waiting for events on contract %v.", self.contractAddress))
						time.Sleep(5000 * time.Millisecond)
					}
				}
			}
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", "")
	return ret_ev_code, err
}

func (self *SolidityContract) Get_contract_address() string {
	return self.contractAddress
}

func (self *SolidityContract) Set_contract_address(addr string) {
	self.contractAddress = addr
	self.filter_id, _ = self.establish_event_listener()
}

func (self *SolidityContract) Get_compiled_contract() *RpcCompiledContract {
	return self.compiledContract
}

func (self *SolidityContract) Set_compiled_contract(con *RpcCompiledContract) {
	self.compiledContract = con
}

func (self *SolidityContract) Set_from(f string) {
	self.from = f
}

func (self *SolidityContract) Set_rpcurl(rpc string) {
	if rpc != "" {
		self.rpcURL = rpc
	}
}

func (self *SolidityContract) Set_skip_eventlistener() {
	self.noEventlistener = true
}

func (self *SolidityContract) is_constant(method_name string) bool {
	function := self.getFunctionFromABI(method_name)
	if function != nil {
		return function.Constant
	} else {
		return false
	}
}

func (self *SolidityContract) get_method_sig(method_name string) (string, error) {
	result, err := "", error(nil)
	function := self.getFunctionFromABI(method_name)
	if function != nil {
		sig := method_name + "("
		for _, inp := range function.Inputs {
			sig += inp.Type + ","
		}
		sig = strings.TrimSuffix(sig, ",") + ")"
		self.logger.Debug("Debug", sig)
		b := []byte(sig)
		result = hex.EncodeToString(b)
	} else {
		err = &FunctionNotFoundError{fmt.Sprintf("Unable to invoke %v because it is not found in the contract interface.\n", method_name)}
	}
	return result, err
}

func (self *SolidityContract) create_contract() (string, error) {
	self.logger.Debug("Entry", "")
	result, out, err := "", "", error(nil)
	var rpcResp *rpcResponse = new(rpcResponse)

	if err = self.check_eth_status(); err == nil {

		params := make(map[string]string)
		params["from"] = self.from
		params["gas"] = "0x16e360"
		params["data"] = self.compiledContract.Code

		if out, err = self.Call_rpc_api("eth_sendTransaction", params); err == nil {
			if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
				if rpcResp.Error.Message != "" {
					err = &RPCError{fmt.Sprintf("RPC contract deploy of %v returned an error: %v.", self.name, rpcResp.Error.Message)}
				} else {
					result = rpcResp.Result.(string)
				}
			}
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", result)
	return result, err
}

func (self *SolidityContract) get_contract(tx_address string) (string, error) {
	self.logger.Debug("Entry", tx_address)
	result, out, found, err := "", "", false, error(nil)
	var rpcResp *rpcGetTransactionResponse = new(rpcGetTransactionResponse)

	start_timer := time.Now()
	for !found && err == nil {
		if out, err = self.Call_rpc_api("eth_getTransactionReceipt", tx_address); err == nil {
			if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
				if rpcResp.Error.Message != "" {
					err = &RPCError{fmt.Sprintf("RPC transaction receipt for deploy of %v returned an error: %v.", self.name, rpcResp.Error.Message)}
				} else {
					//self.logger.Debug("Debug",rpcResp.Result.ContractAddress)
					if rpcResp.Result.ContractAddress != "" {
						result = rpcResp.Result.ContractAddress
						found = true
						update_block(rpcResp.Result.BlockNumber)
						self.log_stats(rpcResp)
					} else {
						delta := time.Now().Sub(start_timer).Seconds()
						if int(delta) < self.tx_delay_toleration {
							self.logger.Debug("Debug", fmt.Sprintf("Waiting for transaction %v to run for %v seconds.", tx_address, delta))
							time.Sleep(5000 * time.Millisecond)
							err = self.check_eth_status()
						} else {
							err = &RPCError{fmt.Sprintf("RPC transaction receipt timed out for tx %v, after %v seconds.", tx_address, delta)}
						}
					}
				}
			}
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", result)
	return result, err
}

func (self *SolidityContract) log_stats(rpcResp *rpcGetTransactionResponse) {
	// If logging blockchain stats, dump them to the log
	if self.logBlockchainStats != "" {
		block_num := rpcResp.Result.BlockNumber
		if out, err := self.Call_rpc_api("eth_getBlockByNumber", &MultiValueParams{block_num, false}); err != nil {
			self.logger.Debug("Error", err.Error())
			return
		} else {
			var rpcResp *rpcGetBlockByNumberResponse = new (rpcGetBlockByNumberResponse)
			if err := json.Unmarshal([]byte(out), rpcResp); err == nil {
				if rpcResp.Error.Message == "" {
					gasUsed, _ := strconv.ParseUint(rpcResp.Result.GasUsed[2:], 16, 64)
					fmt.Printf("Gas used: %v\n", gasUsed)
				}
			}
		}
	}
}

func (self *SolidityContract) establish_event_listener() (string, error) {
	self.logger.Debug("Entry", "")
	result, out, err := "", "", error(nil)
	var rpcResp *rpcResponse = new(rpcResponse)

	if self.noEventlistener == false {
		params := make(map[string]string)
		params["address"] = self.contractAddress

		if out, err = self.Call_rpc_api("eth_newFilter", params); err == nil {
			if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
				if rpcResp.Error.Message != "" {
					err = &RPCError{fmt.Sprintf("RPC contract deploy of %v returned an error: %v.", self.name, rpcResp.Error.Message)}
				} else {
					result = rpcResp.Result.(string)
				}
			}
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", result)
	return result, err
}

func (self *SolidityContract) compile_contract(contract_string string) (*RpcCompiledContract, error) {
	self.logger.Debug("Entry", contract_string[:40]+"...")
	err := error(nil)
	var out string
	var result *RpcCompiledContract

	if jBytes, err := self.get_precompiled_json(); err != nil {
		self.logger.Debug("Debug", fmt.Sprintf("Error reading precompiled json file for %v, %v.", self.name, err))

		// Falling back to use the solidity compiler on demand
		var rpcResp *rpcCompilerResponse = new(rpcCompilerResponse)

		if out, err = self.Call_rpc_api("eth_compileSolidity", contract_string); err == nil {
			if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
				if rpcResp.Error.Message != "" {
					err = &RPCError{fmt.Sprintf("RPC compile invocation of %v returned an error: %v.", self.name, rpcResp.Error.Message)}
				} else {
					if _, ok := rpcResp.Result[self.name]; !ok {
						self.logger.Debug("Debug", rpcResp)
						err = &RPCError{fmt.Sprintf("RPC compile invocation of did not return output for %v.", self.name)}
					} else {
						//self.logger.Debug("Debug",rpcResp.Result[self.name])
						r := rpcResp.Result[self.name]
						self.capture_compiled_json(rpcResp.Result[self.name])
						result = &r
					}
				}
			}
		}

	} else {
		result = new(RpcCompiledContract)
		if err = json.Unmarshal(jBytes, result); err != nil {
			err = &RPCError{fmt.Sprintf("RPC decode of precompiled contract %v returned an error: %v.", self.name, err)}
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", result)
	return result, err
}

func (self *SolidityContract) get_precompiled_json() ([]byte,error) {
	json_file := self.name + ".json"
	json_path := os.Getenv("mtn_contractpath")
	if  json_path == "" {
		json_path = os.Getenv("GOPATH") + "/src/repo.hovitos.engineering/MTN/go-solidity/contracts/" + json_file
	} else {
		json_path += json_file
	}
	jBytes, err := ioutil.ReadFile(json_path)
	return jBytes, err
}

func (self *SolidityContract) capture_compiled_json(rpccc RpcCompiledContract) error {
	if self.integration_test != 0 {
		if jsonBytes, err := json.Marshal(rpccc); err != nil {
			self.logger.Debug("Debug", fmt.Sprintf("Error demarshalling compiled output to json file, %v.", err))
			return err
		} else if err := ioutil.WriteFile(self.name+".json",jsonBytes,0644); err != nil {
			self.logger.Debug("Debug", fmt.Sprintf("Error writing compiled json to a file for %v, %v.", self.name, err))
		}
	}
	return nil
}

func (self *SolidityContract) get_contract_as_string() (string, error) {
	self.logger.Debug("Entry", "")
	contract_string, cBytes, con_file, con_path, err := "", []byte{}, "", "", error(nil)

	con_file = self.name + ".sol"
	con_path = os.Getenv("mtn_contractpath")
	if  con_path == "" {
		con_path = os.Getenv("GOPATH") + "/src/repo.hovitos.engineering/MTN/go-solidity/contracts/" + con_file
	} else {
		con_path += con_file
	}
	cBytes, err = ioutil.ReadFile(con_path)
	contract_string = string(cBytes)

	lines := strings.Split(contract_string, "\n")
	contract_string = ""

	for _, l := range lines {
		if strings.Contains(l, "//") {
			ix := strings.Index(l, "//")
			l = l[:ix]
		}
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, "//") {
			contract_string = contract_string + l
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", contract_string[:40]+"...")
	return contract_string, err
}

func (self *SolidityContract) getFunctionFromABI(methodName string) *abiDefEntry {
	//self.logger.Debug("Entry",methodName)
	abi := self.compiledContract.Info.Abidefinition
	var returnFunction *abiDefEntry
	for _, value := range abi {
		if value.Type == "function" && value.Name == methodName {
			returnFunction = &value
			break
		}
	}
	//self.logger.Debug("Exit ",returnFunction.print())
	return returnFunction
}

func (self *SolidityContract) Call_rpc_api(method string, params interface{}) (string, error) {
	self.logger.Debug("Entry", method, params)
	out, err := "", error(nil)
	var req *http.Request
	var resp *http.Response
	var jsonBytes []byte
	var outBytes []byte
	//var the_params [5]interface{}
	the_params := make([]interface{}, 0, 5)

	body := make(map[string]interface{})
	body["jsonrpc"] = "2.0"
	body["id"] = "1"
	body["method"] = method

	switch params.(type) {
	case *MultiValueParams:
		the_params = append(the_params, params.(*MultiValueParams).A)
		the_params = append(the_params, params.(*MultiValueParams).B)
	default:
		the_params = append(the_params, params)
		if method == "eth_call" {
			the_params = append(the_params, self.get_stable_block())
		}
	}

	// the_params[0] = params
	body["params"] = the_params
	jsonBytes, err = json.Marshal(body)

	self.logger.Debug("Debug", fmt.Sprintf("RPC JSON:%v", body))

	if err == nil {
		req, err = http.NewRequest("POST", self.rpcURL, bytes.NewBuffer(jsonBytes))
		if err == nil {
			req.Close = true			// work around to ensure that Go doesn't get connections confused. Supposed to be fixed in Go 1.6.
			client := &http.Client{}
			resp, err = client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if outBytes, err = ioutil.ReadAll(resp.Body); err == nil {
					out = string(outBytes)
				} else {
					err = &RPCError{fmt.Sprintf("RPC invocation of %v failed reading response message, error: %v", method, outBytes, err.Error())}
				}
			} else {
				err = &RPCError{fmt.Sprintf("RPC http invocation of %v returned error: %v", method, err.Error())}
			}
		} else {
			err = &RPCError{fmt.Sprintf("RPC invocation of %v failed creating http request, error: %v", method, err.Error())}
		}
	} else {
		err = &RPCError{fmt.Sprintf("RPC invocation of %v failed creating JSON body %v, error: %v", method, body, err.Error())}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}

	self.logger.Debug("Exit ", out)
	return out, err
}

func (self *SolidityContract) decodeOutputString(methodName string, output_string string) (interface{}, error) {
	self.logger.Debug("Entry", methodName, output_string)
	err := error(nil)
	var value interface{}

	function := self.getFunctionFromABI(methodName)
	if function != nil {
		for _, outp := range function.Outputs {
			if len(output_string) < 64 {
				err = &UnsupportedValueError{fmt.Sprintf("Unable to decode output from %v because the output is not long enough.", methodName)}
			} else if outp.Type == "address" {
				_, value, err = self.decode_address(methodName, output_string)
			} else if outp.Type == "address[]" {
				_, value, err = self.decode_address_array(methodName, output_string)
			} else if outp.Type == "bool" {
				_, value, err = self.decode_boolean(methodName, output_string)
			} else if outp.Type == "uint256" {
				_, value, err = self.decode_uint256(methodName, output_string)
			} else if outp.Type == "uint256[]" {
				_, value, err = self.decode_uint256_array(methodName, output_string)
			} else if outp.Type == "int256" {
				_, value, err = self.decode_int256(methodName, output_string)
			} else if outp.Type == "int256[]" {
				_, value, err = self.decode_int256_array(methodName, output_string)
			} else if outp.Type == "string" {
				_, value, err = self.decode_string(methodName, output_string)
			} else if outp.Type == "bytes32" {
				_, value, err = self.decode_bytes32(methodName, output_string)
			} else if outp.Type == "bytes32[]" {
				_, value, err = self.decode_bytes32_array(methodName, output_string)
			} else {
				err = &UnsupportedTypeError{fmt.Sprintf("Unable to decode output from %v because type %v is not supported yet. Call Booz.", methodName, outp.Type)}
			}
		}
	} else {
		err = &FunctionNotFoundError{fmt.Sprintf("Unable to decode output from %v because it is not found in the contract interface.\n", methodName)}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", value)
	return value, err
}

func (self *SolidityContract) decode_uint256(methodName string, encoded_output string) (string, uint64, error) {
	self.logger.Debug("Entry", methodName, encoded_output)
	remaining_output, err := "", error(nil)
	var num uint64

	if len(encoded_output) < 64 {
		err = &UnsupportedValueError{fmt.Sprintf("Unable to decode uint256 output from %v because the output is %v bytes long.", methodName, len(encoded_output))}
	} else {
		remaining_output = encoded_output[64:]
		if num, err = strconv.ParseUint(encoded_output[:64], 16, 32); err != nil {
			err = &UnsupportedValueError{fmt.Sprintf("Unable to decode output from %v because parameter %v is not a number. Internal error: %v", methodName, encoded_output[:64], err)}
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", remaining_output, num)
	return remaining_output, num, err
}

func (self *SolidityContract) decode_int256(methodName string, encoded_output string) (string, uint64, error) {
	return self.decode_uint256(methodName, encoded_output)
}

func (self *SolidityContract) decode_string(methodName string, encoded_output string) (string, string, error) {
	self.logger.Debug("Entry", methodName, encoded_output)
	out, remaining_output, value, err := "", "", "", error(nil)
	var length uint64
	var b []byte

	if out, length, err = self.decode_uint256(methodName, encoded_output[64:]); err == nil {
		if uint64(len(out)) < length {
			err = &UnsupportedValueError{fmt.Sprintf("Unable to decode string output from %v because the output is shorter than required. Need %v, have %v.", methodName, length, len(out))}
		} else {
			if b, err = hex.DecodeString(out[:length*2]); err == nil {
				value = string(b)
				remaining_output = out[length*2:]
			} else {
				err = &UnsupportedValueError{fmt.Sprintf("Unable to decode string output from %v because the output %v is not a string. Internal error: %v", methodName, out[:length*2], err)}
			}
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", remaining_output, value)
	return remaining_output, value, err
}

func (self *SolidityContract) decode_bytes32(methodName string, encoded_output string) (string, string, error) {
	self.logger.Debug("Entry", methodName, encoded_output)
	out, remaining_output, value, err := "", "", "", error(nil)
	var b []byte


	if len(encoded_output) < 64 {
		err = &UnsupportedValueError{fmt.Sprintf("Unable to decode bytes32 output from %v because the output is shorter than required. Need %v, have %v.", methodName, 64, len(encoded_output))}
	} else {
		out = encoded_output[:64]
		if b, err = hex.DecodeString(out); err == nil {
			value = string(b)
			remaining_output = encoded_output[64:]
		} else {
			err = &UnsupportedValueError{fmt.Sprintf("Unable to decode string output from %v because the output %v is not a string. Internal error: %v", methodName, encoded_output[:64], err)}
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", remaining_output, value)
	return remaining_output, value, err
}

func (self *SolidityContract) decode_string_array(methodName string, encoded_output string) (string, []string, error) {
	self.logger.Debug("Entry", methodName, encoded_output)
	out, val, remaining_output, err := "", "", "", error(nil)
	var length uint64
	var value []string

	if len(encoded_output) < 64*2 {
		err = &UnsupportedValueError{fmt.Sprintf("Unable to decode output from %v because the output is shorter than required. Need %v, have %v.", methodName, 64*2, len(encoded_output))}
	} else {
		if out, length, err = self.decode_uint256(methodName, encoded_output[64:]); err == nil {
			if uint64(len(out)) < length*128 {
				err = &UnsupportedValueError{fmt.Sprintf("Unable to decode string array output from %v because the output is shorter than required. Need %v, have %v.", methodName, length*64, len(out))}
			} else {
				value = make([]string, 0, length)
				var i uint64
				for i = 0; i < length; i++ {
					if out, val, err = self.decode_string(methodName, out); err == nil {
						value = append(value, val)
						remaining_output = out
					} else {
						break
					}
				}
			}
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", remaining_output, value)
	return remaining_output, value, err
}

func (self *SolidityContract) decode_bytes32_array(methodName string, encoded_output string) (string, []string, error) {
	self.logger.Debug("Entry", methodName, encoded_output)
	out, val, remaining_output, err := "", "", "", error(nil)
	var length uint64
	var value []string

	if len(encoded_output) < 64*2 {
		err = &UnsupportedValueError{fmt.Sprintf("Unable to decode output from %v because the output is shorter than required. Need %v, have %v.", methodName, 64*2, len(encoded_output))}
	} else {
		if out, length, err = self.decode_uint256(methodName, encoded_output[64:]); err == nil {
			if uint64(len(out)) < length*64 {
				err = &UnsupportedValueError{fmt.Sprintf("Unable to decode string array output from %v because the output is shorter than required. Need %v, have %v.", methodName, length*64, len(out))}
			} else {
				value = make([]string, 0, length)
				var i uint64
				for i = 0; i < length; i++ {
					if out, val, err = self.decode_bytes32(methodName, out); err == nil {
						value = append(value, val)
						remaining_output = out
					} else {
						break
					}
				}
			}
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", remaining_output, value)
	return remaining_output, value, err
}

func (self *SolidityContract) decode_address_array(methodName string, encoded_output string) (string, []string, error) {
	self.logger.Debug("Entry", methodName, encoded_output)
	out, val, remaining_output, err := "", "", "", error(nil)
	var length uint64
	var value []string

	if len(encoded_output) < 64*2 {
		err = &UnsupportedValueError{fmt.Sprintf("Unable to decode output from %v because the output is shorter than required. Need %v, have %v.", methodName, 64*2, len(encoded_output))}
	} else {
		if out, length, err = self.decode_uint256(methodName, encoded_output[64:]); err == nil {
			if uint64(len(out)) < length*64 {
				err = &UnsupportedValueError{fmt.Sprintf("Unable to decode address array output from %v because the output is shorter than required. Need %v, have %v.", methodName, length*64, len(out))}
			} else {
				value = make([]string, 0, length)
				var i uint64
				for i = 0; i < length; i++ {
					if out, val, err = self.decode_address(methodName, out); err == nil {
						value = append(value, val)
						remaining_output = out
					} else {
						break
					}
				}
			}
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", remaining_output, value)
	return remaining_output, value, err
}

func (self *SolidityContract) decode_int256_array(methodName string, encoded_output string) (string, []uint64, error) {
	return self.decode_uint256_array(methodName, encoded_output)
}

func (self *SolidityContract) decode_uint256_array(methodName string, encoded_output string) (string, []uint64, error) {
	self.logger.Debug("Entry", methodName, encoded_output)
	out, remaining_output, value, err := "", "", []uint64{}, error(nil)
	var length, val uint64

	if len(encoded_output) < 64*2 {
		err = &UnsupportedValueError{fmt.Sprintf("Unable to decode uint256 array output from %v because the output is shorter than required. Need %v, have %v.", methodName, 64*2, len(encoded_output))}
	} else {
		if out, length, err = self.decode_uint256(methodName, encoded_output[64:]); err == nil {
			if uint64(len(out)) < length*64 {
				err = &UnsupportedValueError{fmt.Sprintf("Unable to decode uint256 array output from %v because the output is shorter than required. Need %v, have %v.", methodName, length*64, len(out))}
			} else {
				value = make([]uint64, 0, length)
				var i uint64
				for i = 0; i < length; i++ {
					if out, val, err = self.decode_uint256(methodName, out); err == nil {
						value = append(value, val)
						remaining_output = out
					} else {
						break
					}
				}
			}
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", remaining_output, value)
	return remaining_output, value, err
}

func (self *SolidityContract) decode_address(methodName string, encoded_output string) (string, string, error) {
	self.logger.Debug("Entry", methodName, encoded_output)
	remaining_output, value, err := "", "", error(nil)

	remaining_output = encoded_output[64:]
	value = "0x" + encoded_output[24:64]

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", remaining_output, value)
	return remaining_output, value, err
}

func (self *SolidityContract) decode_boolean(methodName string, encoded_output string) (string, bool, error) {
	self.logger.Debug("Entry", methodName, encoded_output)
	remaining_output, value := "", false

	remaining_output = encoded_output[64:]
	if encoded_output[63] == []byte("0")[0] {
		value = false
	} else {
		value = true
	}

	self.logger.Debug("Exit ", remaining_output, value)
	return remaining_output, value, nil
}

func (self *SolidityContract) encodeInputString(methodName string, params []interface{}) (string, error) {
	self.logger.Debug("Entry", methodName, params)
	err := error(nil)
	res := ""
	param_string := ""
	param_string_front := ""
	param_string_back := ""
	function := self.getFunctionFromABI(methodName)
	if function != nil {
		for index, inp := range function.Inputs {
			if inp.Type == "uint256" {
				if res, err = self.encode_uint256(methodName, params[index]); err == nil {
					param_string_front += res
				}
			} else if inp.Type == "int256" {
				if res, err = self.encode_int256(methodName, params[index]); err == nil {
					param_string_front += res
				}
			} else if inp.Type == "bool" {
				if res, err = self.encode_boolean(methodName, params[index]); err == nil {
					param_string_front += res
				}
			} else if inp.Type == "string" {
				if res, err = self.encode_uint256(methodName, len(function.Inputs)*32+len(param_string_back)/2); err == nil {
					param_string_front += res
					if res, err = self.encode_string(methodName, params[index]); err == nil {
						param_string_back += res
					}
				}
			} else if inp.Type == "bytes32" {
				if res, err = self.encode_bytes32(methodName, params[index]); err == nil {
					param_string_front += res
				}
			} else if inp.Type == "bytes32[]" {
				if res, err = self.encode_uint256(methodName, len(function.Inputs)*32+len(param_string_back)/2); err == nil {
					param_string_front += res
					if res, err = self.encode_bytes32_array(methodName, params[index]); err == nil {
						param_string_back += res
					}
				}
			} else if inp.Type == "address" {
				if res, err = self.encode_address(methodName, params[index]); err == nil {
					param_string_front += res
				}
			} else if inp.Type == "uint256[]" {
				if res, err = self.encode_uint256(methodName, len(function.Inputs)*32+len(param_string_back)/2); err == nil {
					param_string_front += res
					if res, err = self.encode_uint256_array(methodName, params[index]); err == nil {
						param_string_back += res
					}
				}
			} else {
				err = &UnsupportedTypeError{fmt.Sprintf("Unable to invoke %v because type %v is not supported yet. Call Booz.", methodName, inp.Type)}
			}
		}
		param_string = param_string_front + param_string_back
	} else {
		err = &FunctionNotFoundError{fmt.Sprintf("Unable to invoke %v because it is not found in the contract interface.\n", methodName)}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", param_string)
	return param_string, err

}

func (self *SolidityContract) encode_uint256(methodName string, param interface{}) (string, error) {
	self.logger.Debug("Entry", methodName, param)
	err := error(nil)
	strVal := ""
	num := 0

	switch param.(type) {
	case int:
		num = param.(int)
	case string:
		if num, err = strconv.Atoi(param.(string)); err != nil {
			err = &UnsupportedValueError{fmt.Sprintf("Unable to invoke %v because parameter %v is not a number. Internal error: %v", methodName, param, err)}
		}
	case nil:
		err = &UnsupportedTypeError{fmt.Sprintf("Unable to invoke %v because parameter is nil.", methodName)}
	default:
		err = &UnsupportedTypeError{fmt.Sprintf("Unable to invoke %v because parameter %v is not a string or integer, is %v.", methodName, param, reflect.TypeOf(param).String())}
	}

	if err == nil {
		if num < 0 {
			err = &UnsupportedValueError{fmt.Sprintf("Unable to invoke %v because parameter %v is negative.", methodName, num)}
		} else {
			strVal = fmt.Sprintf("%x", num)
			strVal = self.zero_pad_left(strVal, 64)
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", strVal)
	return strVal, err
}

func (self *SolidityContract) encode_int256(methodName string, param interface{}) (string, error) {
	return self.encode_uint256(methodName, param)
}

func (self *SolidityContract) encode_boolean(methodName string, param interface{}) (string, error) {
	self.logger.Debug("Entry", methodName, param)
	encoding, err, value := "", error(nil), false

	switch param.(type) {
	case bool:
		value = param.(bool)

	default:
		err = &UnsupportedTypeError{fmt.Sprintf("Unable to invoke %v because parameter %v is not a boolean.", methodName, param)}
	}

	if err == nil {
		encoding = strings.Repeat("0", 63)
		if value == true {
			encoding += "1"
		} else {
			encoding += "0"
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", encoding)
	return encoding, err
}

func (self *SolidityContract) encode_string(methodName string, param interface{}) (string, error) {
	self.logger.Debug("Entry", methodName, param)
	encoding, res, err, value := "", "", error(nil), ""

	switch param.(type) {
	case string:
		value = param.(string)

	default:
		err = &UnsupportedTypeError{fmt.Sprintf("Unable to invoke %v because parameter %v is not a string.", methodName, param)}
	}

	if err == nil {
		left_over := len(value) % 32
		b := []byte(value)
		val := hex.EncodeToString(b)
		if res, err = self.encode_uint256(methodName, len(value)); err == nil {
			encoding += res
			encoding += val + strings.Repeat("0", ((32-left_over)*2))
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", encoding)
	return encoding, err
}

func (self *SolidityContract) encode_address(methodName string, param interface{}) (string, error) {
	self.logger.Debug("Entry", methodName, param)
	encoding, err, value := "", error(nil), ""

	switch param.(type) {
	case string:
		str := param.(string)
		if len(str) == 40 || len(str) == 42 {
			if strings.HasPrefix(str, "0x") {
				value = str[2:]
			} else {
				value = str
			}
			value = self.zero_pad_left(value, 64)
		} else {
			err = &UnsupportedValueError{fmt.Sprintf("Unable to invoke %v because parameter %v is not a 40 byte string.", methodName, param)}
		}
	default:
		err = &UnsupportedTypeError{fmt.Sprintf("Unable to invoke %v because parameter %v is not a string.", methodName, param)}
	}

	if err == nil {
		encoding += value
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", encoding)
	return encoding, err
}

func (self *SolidityContract) encode_bytes32(methodName string, param interface{}) (string, error) {
	self.logger.Debug("Entry", methodName, param)
	encoding, err, value := "", error(nil), ""

	switch param.(type) {
	case string:
		str := param.(string)
		if len(str) <= 32 {
			b := []byte(str)
			value = hex.EncodeToString(b)
			encoding += value + strings.Repeat("0", ((32-len(str))*2))
		} else {
			err = &UnsupportedValueError{fmt.Sprintf("Unable to invoke %v because parameter %v is larger than a 32 byte string.", methodName, param)}
		}
	default:
		err = &UnsupportedTypeError{fmt.Sprintf("Unable to invoke %v because parameter %v is not a string.", methodName, param)}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", encoding)
	return encoding, err
}

func (self *SolidityContract) encode_bytes32_array(methodName string, param interface{}) (string, error) {
	self.logger.Debug("Entry", methodName, param)
	encoding, res, err, str := "", "", error(nil), []string{}

	switch param.(type) {
	case []string:
		str = param.([]string)

	default:
		err = &UnsupportedTypeError{fmt.Sprintf("Unable to invoke %v because parameter %v is not a string array.", methodName, param)}
	}

	if err == nil {
		if res, err = self.encode_uint256(methodName, len(str)); err == nil {
			encoding += res
			for _, v := range str {
				if res, err = self.encode_bytes32(methodName, v); err == nil {
					encoding += res
				} else {
					break
				}
			}
		}
	}
	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", encoding)
	return encoding, err
}

func (self *SolidityContract) encode_uint256_array(methodName string, param interface{}) (string, error) {
	self.logger.Debug("Entry", methodName, param)
	encoding, res, err, value := "", "", error(nil), []int{}

	switch param.(type) {
	case []int:
		value = param.([]int)

	default:
		err = &UnsupportedTypeError{fmt.Sprintf("Unable to invoke %v because parameter %v is not an array of integers.", methodName, param)}
	}

	if err == nil {
		if res, err = self.encode_uint256(methodName, len(value)); err == nil {
			encoding += res
			for _, v := range value {
				if res, err = self.encode_uint256(methodName, v); err == nil {
					encoding += res
				} else {
					break
				}
			}
		}
	}

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ", encoding)
	return encoding, err
}

func (self *SolidityContract) zero_pad_left(p string, length int) string {
	if len(p)%length == 0 {
		return p
	} else {
		if len(p) < length {
			return strings.Repeat("0", (length-len(p))) + p
		} else {
			return strings.Repeat("0", (length-(len(p)-((len(p)/length)*length)))) + p
		}
	}
}

func (self *SolidityContract) get_current_block() (string, error) {
	var rpcResp *rpcResponse = new(rpcResponse)
	if res,err := self.Call_rpc_api("eth_blockNumber",nil); err != nil {
        err = &RPCError{fmt.Sprintf("RPC invocation of eth_blockNumber returned an error: %v.",err)}
        return "", err
    } else if err := json.Unmarshal([]byte(res),rpcResp); err != nil {
        err = &RPCError{fmt.Sprintf("RPC invocation of eth_blockNumber returned undecodable response %v, error: %v.",res,err)}
        return "", err
    } else if rpcResp.Error.Message != "" {
        err = &RPCError{fmt.Sprintf("RPC invocation of eth_blockNumber returned an error: %v.",rpcResp.Error.Message)}
        return "", err
    } else {
    	switch rpcResp.Result.(type) {
        case string:
            if rpcResp.Result != "0x0" {
                update_block(rpcResp.Result.(string))
            }
            return rpcResp.Result.(string), nil
        default:
        	err = &RPCError{fmt.Sprintf("RPC invocation of eth_blockNumber returned result that is not a string: %v.",rpcResp.Result)}
        	return "", err
        }
    }
}


func (self *SolidityContract) check_eth_status() error {
	self.logger.Debug("Entry")
	err := error(nil)
	var res string
    var rpcResp *rpcResponse = new(rpcResponse)
    net_done := false
    block_done := false
    sync_done := false

    poll_wait := 5

    start_timer := time.Now()
    for !net_done && self.integration_test == 0 {
        if res,err = self.Call_rpc_api("net_peerCount",nil); err != nil {
            err = &RPCError{fmt.Sprintf("RPC invocation of net_peerCount returned an error: %v.",err)}
            break
        }
        if err = json.Unmarshal([]byte(res),rpcResp); err != nil {
            err = &RPCError{fmt.Sprintf("RPC invocation of net_peerCount returned undecodable response %v, error: %v.",res,err)}
            break
        }

        if rpcResp.Error.Message != "" {
            err = &RPCError{fmt.Sprintf("RPC invocation of net_peerCount returned an error: %v.",rpcResp.Error.Message)}
            break
        } else {
            switch rpcResp.Result.(type) {
                case string:
                    if rpcResp.Result != "0x0" {
                        net_done = true
                        break
                    }
                default:
            }
            if net_done {
            	break
            }
            delta := time.Now().Sub(start_timer).Seconds()
			if int(delta) < self.sync_delay_toleration {
				time.Sleep(time.Duration(poll_wait)*1000*time.Millisecond)
				self.logger.Debug("Debug", fmt.Sprintf("Waiting for non-zero peer count for %v seconds.", delta))
			} else {
				err = &RPCError{fmt.Sprintf("Peer count check timed out, after %v seconds.", delta)}
				break
			}
        }
    }

    start_timer = time.Now()
    block := ""
    for !block_done && err == nil {
    	if block, err = self.get_current_block(); err != nil {
    		break
        // if res,err = self.Call_rpc_api("eth_blockNumber",nil); err != nil {
        //     err = &RPCError{fmt.Sprintf("RPC invocation of eth_blockNumber returned an error: %v.",err)}
        //     break
        // }
        // if err = json.Unmarshal([]byte(res),rpcResp); err != nil {
        //     err = &RPCError{fmt.Sprintf("RPC invocation of eth_blockNumber returned undecodable response %v, error: %v.",res,err)}
        //     break
        // }

        // if rpcResp.Error.Message != "" {
        //     err = &RPCError{fmt.Sprintf("RPC invocation of eth_blockNumber returned an error: %v.",rpcResp.Error.Message)}
        //     break
        } else if block != "0x0" {
            // switch brpcResp.Result.(type) {
            //     case string:
            //         if brpcResp.Result != "0x0" {
                        block_done = true
                        // update_block(brpcResp.Result.(string))
                        break
                //     }
                // default:
            // }
            // if block_done {
            // 	break
            // }
        } else {
            delta := time.Now().Sub(start_timer).Seconds()
			if int(delta) < self.sync_delay_toleration {
				time.Sleep(time.Duration(poll_wait)*1000*time.Millisecond)
				self.logger.Debug("Debug", fmt.Sprintf("Waiting for non-zero block count for %v seconds.", delta))
			} else {
				err = &RPCError{fmt.Sprintf("Block count check timed out, after %v seconds.", delta)}
				break
			}
        }
    }

    start_timer = time.Now()
    for !sync_done && err == nil {
        if res,err = self.Call_rpc_api("eth_syncing",nil); err == nil {
            if err = json.Unmarshal([]byte(res),rpcResp); err == nil {
                if rpcResp.Error.Message != "" {
                    err = &RPCError{fmt.Sprintf("RPC invocation of eth_syncing returned an error: %v.",rpcResp.Error.Message)}
                    break
                } else {
                    switch rpcResp.Result.(type) {
                        case bool:
                            if rpcResp.Result == false {
                                sync_done = true
                                break
                            }
                        default:
                    }
                    if sync_done {
            			break
            		}
                    delta := time.Now().Sub(start_timer).Seconds()
					if int(delta) < self.sync_delay_toleration {
						time.Sleep(time.Duration(poll_wait)*1000*time.Millisecond)
						self.logger.Debug("Debug", fmt.Sprintf("Waiting for syncing to complete for %v seconds.", delta))
					} else {
						err = &RPCError{fmt.Sprintf("Sync check timed out, after %v seconds.", delta)}
						break
					}
                }
            } else {
                err = &RPCError{fmt.Sprintf("RPC invocation of eth_syncing returned undecodable response %v, error: %v.",res,err)}
            	break
            }
        } else {
            err = &RPCError{fmt.Sprintf("RPC invocation of eth_syncing returned an error: %v.",err)}
            break
        }
    }

    if err == nil {
        if res,err = self.Call_rpc_api("eth_getBalance",&MultiValueParams{self.from, "latest"}); err == nil {
	        if err = json.Unmarshal([]byte(res),rpcResp); err == nil {
	            if rpcResp.Error.Message != "" {
	                err = &RPCError{fmt.Sprintf("RPC invocation of eth_getBalance returned an error: %v.",rpcResp.Error.Message)}
	            } else {
	                switch rpcResp.Result.(type) {
	                    case string:
							bal := big.NewInt(0)
							bal_hex_str := rpcResp.Result.(string)
					        // the math/big library doesn't like leading "0x" on hex strings
					        bal.SetString(bal_hex_str[2:],16)
	                        if bal.Cmp(big.NewInt(1500000)) < 1 {
	                            err = &RPCError{fmt.Sprintf("Out of ether, have: %v.",rpcResp.Result)}
	                        }
	                    default:
	                }

	            }
	        } else {
	            err = &RPCError{fmt.Sprintf("RPC invocation of eth_getBalance returned undecodable response %v, error: %v.",res,err)}
	        }
	    } else {
	        err = &RPCError{fmt.Sprintf("RPC invocation of eth_getBalance returned an error: %v.",err)}
	    }
	}

    if err == nil && blocks_stopped() {
        err = &RPCError{fmt.Sprintf("No new blocks received in last %v seconds. Last block was %v.", no_recent_blocks, global_block_state.blockNumber)}
    }

    self.logger.Debug("Debug", fmt.Sprintf("Global block %v", global_block_state))

	if err != nil {
		self.logger.Debug("Error", err.Error())
	}
	self.logger.Debug("Exit ")
	return err
}



// ============================================================================
// Errors surfaced by this class
//

type FunctionNotFoundError struct {
	msg string
}

func (e *FunctionNotFoundError) Error() string {
	if e != nil {
		return e.msg
	} else {
		return ""
	}
}

type UnsupportedTypeError struct {
	msg string
}

func (e *UnsupportedTypeError) Error() string {
	if e != nil {
		return e.msg
	} else {
		return ""
	}
}

type UnsupportedValueError struct {
	msg string
}

func (e *UnsupportedValueError) Error() string {
	if e != nil {
		return e.msg
	} else {
		return ""
	}
}

type RPCError struct {
	msg string
}

func (e *RPCError) Error() string {
	if e != nil {
		return e.msg
	} else {
		return ""
	}
}

type DeployError struct {
	msg string
}

func (e *DeployError) Error() string {
	if e != nil {
		return e.msg
	} else {
		return ""
	}
}

type LoadError struct {
	msg string
}

func (e *LoadError) Error() string {
	if e != nil {
		return e.msg
	} else {
		return ""
	}
}

// ============================================================================
// Structs returned by the compiler RPC
//

type rpcResponse struct {
	Id      string      `json:"id"`
	Version string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	Error   struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type rpcGetBlockByNumberResponse struct {
	Id      string     `json:"id"`
	Version string     `json:"jsonrpc"`
	Result  rpcBlock   `json:"result"`
	Error   struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type rpcBlock struct {
	Number           string   `json:"number"`
	Hash             string   `json:"hash"`
	ParentHash       string   `json:"parentHash"`
	Nonce            string   `json:"nonce"`
	Sha3Uncles       string   `json:"sha3Uncles"`
	LogsBloom        string   `json:"logsBloom"`
	TransactionsRoot string   `json:"transactionsRoot"`
	StateRoot        string   `json:"stateRoot"`
	ReceiptRoot      string   `json:"receiptRoot"`
	Miner            string   `json:"miner"`
	Difficulty       string   `json:"difficulty"`
	TotalDifficulty  string   `json:"totalDifficulty"`
	ExtraData        string   `json:"extraData"`
	Size             string   `json:"size"`
	GasLimit         string   `json:"gasLimit"`
	GasUsed          string   `json:"gasUsed"`
	Timestamp        string   `json:"timestamp"`
	Transactions   []string   `json:"transactions"`
	Uncles         []string   `json:"uncles"`
}

type rpcGetTransactionResponse struct {
	Id      string         `json:"id"`
	Version string         `json:"jsonrpc"`
	Result  rpcTranReceipt `json:"result"`
	Error   struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type rpcTranReceipt struct {
	TransactionHash   string   `json:"transactionHash"`
	Transactionindex  string   `json:"transactionIndex"`
	BlockNumber       string   `json:"blockNumber"`
	BlockHash         string   `json:"blockHash"`
	CumulativeGasUsed string   `json:"cumulativeGasUsed"`
	GasUsed           string   `json:"gasUsed"`
	ContractAddress   string   `json:"contractAddress"`
	Logs              []interface{} `json:"logs"`
}

type rpcGetFilterChangesResponse struct {
	Id      string             `json:"id"`
	Version string             `json:"jsonrpc"`
	Result  []rpcFilterChanges `json:"result"`
	Error   struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type rpcFilterChanges struct {
	LogIndex         string   `json:"logIndex"`
	TransactionHash  string   `json:"transactionHash"`
	TransactionIndex string   `json:"transactionIndex"`
	BlockNumber      string   `json:"blockNumber"`
	BlockHash        string   `json:"blockHash"`
	Address          string   `json:"address"`
	Data             string   `json:"data"`
	Topics           []string `json:"topics"`
}

type rpcCompilerResponse struct {
	Id      string                         `json:"id"`
	Version string                         `json:"jsonrpc"`
	Result  map[string]RpcCompiledContract `json:"result"`
	Error   struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type abiDefEntry struct {
	Inputs []struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"inputs"`
	Type     string `json:"type"`
	Constant bool   `json:"constant"`
	Name     string `json:"name"`
	Outputs  []struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"outputs"`
}

func (self *abiDefEntry) print() string {
	if self != nil {
		return self.Name
	} else {
		return ""
	}
}

type RpcCompiledContract struct {
	Info struct {
		Language        string        `json:"language"`
		Languageversion string        `json:"languageVersion"`
		Abidefinition   []abiDefEntry `json:"abiDefinition"`
		Compilerversion string        `json:"compilerVersion"`
		Developerdoc    struct {
			Methods struct {
			} `json:"methods"`
		} `json:"developerDoc"`
		Userdoc struct {
			Methods struct {
			} `json:"methods"`
		} `json:"userDoc"`
		Source string `json:"source"`
	} `json:"info"`
	Code string `json:"code"`
}

type MultiValueParams struct {
	A, B interface{}
}


