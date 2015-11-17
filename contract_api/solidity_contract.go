package contract_api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"repo.hovitos.engineering/MTN/go-solidity/utility"
	"strconv"
	"strings"
	"time"
)

type SolidityContract struct {
	baseBody            map[string]string
	name                string
	from                string
	rpcURL              string
	compiledContract    *rpcCompiledContract
	contractAddress     string
	filter_id           string
	noEventlistener     bool
	tx_delay_toleration int
	logger              *utility.DebugTrace
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
		delay = 60
	}
	sc.tx_delay_toleration = delay
	return sc
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
	out, method_id, invocation_string, eth_method, found, err := "", "", "", "", false, error(nil)
	var result interface{}
	var rpcResp *rpcResponse = new(rpcResponse)

	if hex_sig, err := self.get_method_sig(method_name); err == nil {
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

	if err == nil {
		if invocation_string, err = self.encodeInputString(method_name, params); err == nil {
			invocation_string = method_id + invocation_string
			eth_method = "eth_call"
			if !self.is_constant(method_name) {
				eth_method = "eth_sendTransaction"
			}
		}
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

						start_timer := time.Now()
						for !found && err == nil {
							if out, err = self.Call_rpc_api("eth_getTransactionReceipt", tx_address); err == nil {
								if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
									if rpcResp.Error.Message != "" {
										err = &RPCError{fmt.Sprintf("RPC transaction receipt for tx %v, invoking %v returned an error: %v.", tx_address, method_name, rpcResp.Error.Message)}
									} else {
										//self.logger.Debug("Debug",rpcResp.Result)
										if rpcResp.Result != nil {
											result = 0
											found = true
										} else {
											delta := time.Now().Sub(start_timer).Seconds()
											if int(delta) < self.tx_delay_toleration {
												self.logger.Debug("Debug", fmt.Sprintf("Waiting for transaction %v to run for %v seconds.", tx_address, delta))
												time.Sleep(5000 * time.Millisecond)
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
							result = 0
							// err = &RPCError{fmt.Sprintf("RPC invocation returned %v, the EVM probably failed executing method %v.", rpcResp.Result, method_name)}
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
					} else {
						delta := time.Now().Sub(start_timer).Seconds()
						if int(delta) < self.tx_delay_toleration {
							self.logger.Debug("Debug", fmt.Sprintf("Waiting for transaction %v to run for %v seconds.", tx_address, delta))
							time.Sleep(5000 * time.Millisecond)
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

func (self *SolidityContract) compile_contract(contract_string string) (*rpcCompiledContract, error) {
	self.logger.Debug("Entry", contract_string[:40]+"...")
	err := error(nil)
	var out string
	var result *rpcCompiledContract
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
					result = &r
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

func (self *SolidityContract) get_contract_as_string() (string, error) {
	self.logger.Debug("Entry", "")
	contract_string, cBytes, con_file, con_path, err := "", []byte{}, "", "", error(nil)

	con_file = self.name + ".sol"
	con_path = os.Getenv("GOPATH") + "/src/repo.hovitos.engineering/MTN/go-solidity/contracts/" + con_file
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
	var the_params [1]interface{}

	body := make(map[string]interface{})
	body["jsonrpc"] = "2.0"
	body["id"] = "1"
	body["method"] = method

	the_params[0] = params
	body["params"] = the_params
	jsonBytes, err = json.Marshal(body)
	//self.logger.Debug("Debug",string(jsonBytes))
	if err == nil {
		req, err = http.NewRequest("POST", self.rpcURL, bytes.NewBuffer(jsonBytes))
		if err == nil {
			client := &http.Client{}
			resp, err = client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if outBytes, err = ioutil.ReadAll(resp.Body); err == nil {
					out = string(outBytes)
					//self.logger.Debug("Debug",out)
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

	if out, length, err = self.decode_uint256(methodName, encoded_output[64:]); err == nil {
		if uint64(len(out)) < length {
			err = &UnsupportedValueError{fmt.Sprintf("Unable to decode string output from %v because the output is shorter than required. Need %v, have %v.", methodName, length, len(out))}
		} else {
			if b, err := hex.DecodeString(out[:length*2]); err == nil {
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

	if len(encoded_output) < 64 {
		err = &UnsupportedValueError{fmt.Sprintf("Unable to decode bytes32 output from %v because the output is shorter than required. Need %v, have %v.", methodName, 64, len(encoded_output))}
	} else {
		out = encoded_output[:64]
		if b, err := hex.DecodeString(out); err == nil {
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
	Logs              []string `json:"logs"`
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
	Result  map[string]rpcCompiledContract `json:"result"`
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

type rpcCompiledContract struct {
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
