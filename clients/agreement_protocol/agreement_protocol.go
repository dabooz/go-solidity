package main

import (
    "bytes"
    "fmt"
    "encoding/hex"
    "encoding/json"
    "log"
    "github.com/open-horizon/go-solidity/contract_api"
    "os"
    "strings"
    "time"
    )

func main() {
    fmt.Println("Starting agreements client")

    if len(os.Args) < 3 {
        fmt.Printf("...terminating, only %v parameters were passed.\n",len(os.Args))
        os.Exit(1)
    }

    dir_contract := os.Args[1]
    if !strings.HasPrefix(dir_contract, "0x") {
        dir_contract = "0x" + dir_contract
    }
    fmt.Printf("using directory %v\n",dir_contract)
    agreements_owner := os.Args[2]
    if !strings.HasPrefix(agreements_owner, "0x") {
        agreements_owner = "0x" + agreements_owner
    }
    fmt.Printf("using account %v\n",agreements_owner)

    err := error(nil)

    // Establish the directory contract
    dirc := contract_api.SolidityContractFactory("directory")
    if _,err := dirc.Load_contract(agreements_owner, ""); err != nil {
        fmt.Printf("...terminating, could not load directory contract: %v\n",err)
        os.Exit(1)
    }
    dirc.Set_contract_address(dir_contract)

    // Find the agreements contract
    var agaddr interface{}
    p := make([]interface{},0,10)
    p = append(p,"agreements")
    if agaddr,err = dirc.Invoke_method("get_entry",p); err != nil {
        log.Printf("...terminating, could not find agreements in directory: %v\n",err)
        os.Exit(1)
    }
    log.Printf("agreements addr is %v\n",agaddr)

    // Establish the agreements contract
    ag := contract_api.SolidityContractFactory("agreements")
    if _,err := ag.Load_contract(agreements_owner, ""); err != nil {
        log.Printf("...terminating, could not load agreements contract: %v\n",err)
        os.Exit(1)
    }
    ag.Set_contract_address(agaddr.(string))


    // ===================================================================================================
    // Prepare to make the first create_agreement call with a simple set of parameters and leave it in
    // the system. Here we get a hash of a test string and then sign the string. These are used throughout
    // all the tests.
    //
    fmt.Println("Hash and sign a simple string.")
    smarter_contract := "{long string of contract terms}"
    hash_string := "0x" + hex.EncodeToString([]byte(smarter_contract))
    var rpcResp *rpcResponse = new(rpcResponse)
    sig_hash, sig, out := "", "", ""

    if out, err = ag.Call_rpc_api("web3_sha3", hash_string); err == nil {
        if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
            if rpcResp.Error.Message != "" {
                log.Printf("RPC hash of terms and conditions failed, error: %v.", rpcResp.Error.Message)
                os.Exit(1)
            } else {
                sig_hash = rpcResp.Result.(string)
                log.Printf("Hash of terms and conditions is: %v\n", sig_hash)
            }
        }
    }

    if out, err = ag.Call_rpc_api("eth_sign", &contract_api.MultiValueParams{agreements_owner, sig_hash}); err == nil {
        if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
            if rpcResp.Error.Message != "" {
                log.Printf("RPC sign of terms and conditions hash failed, error: %v.", rpcResp.Error.Message)
                os.Exit(1)
            } else {
                sig = rpcResp.Result.(string)
                log.Printf("Signature of terms and conditions hash is: %v\n", sig)
            }
        }
    }

    if len(sig[2:]) != 130 {
        log.Printf("Signature has wrong length: %v.", len(sig[2:]))
        os.Exit(1)
    }

    // ===================================================================================================
    // This is the heart of the testcase. Here we will start trying to make agreements on the blockchain.
    //
    // Invoke the blockchain to make the first agreement.
    //
    agID := []byte("00000000000000000000000000000000")
    fmt.Printf("Make a simple agreement using %v\n", agID)
    make_agreement(ag, agID, sig_hash, sig, agreements_owner, true)


    // ===================================================================================================
    // Make another create_agreement call and terminate the agreement.
    //
    agID = []byte("11111111111111111111111111111111")
    fmt.Printf("Make a second agreement using ID: %v\n", agID)
    make_agreement(ag, agID, sig_hash, sig, agreements_owner, true)

    terminate_agreement(ag, agID, agreements_owner, true)


    // ===================================================================================================
    // Try to make some agreements with incorrect info to prove that the smart contract will reject
    // invalid agreement attempts or invalid terminations.
    //

    // 1. Use an invalid signature - should not make an agreement, will generate fraud event
    agID = []byte("22222222222222222222222222222222")
    fmt.Printf("Try to make an agreement using an invalid signature, ID:%v\n", agID)
    make_agreement(ag, agID, sig_hash, "012345678901234567890123456789012345678901234567890123456789", agreements_owner, false)


    // 2. Use an existing agreement ID with a different hash - should not make an agreement, 
    agID = []byte("00000000000000000000000000000000")
    fmt.Printf("Try to make an agreement using an invalid hash, ID:%v\n", agID)
    make_agreement(ag, agID, "11111111111111111111111111111111", sig, agreements_owner, false)


    // 3. Pass no counter party - should not make an agreement
    agID = []byte("22222222222222222222222222222222")
    fmt.Printf("Try to make an agreement without providing a counterParty, ID:%v\n", agID)
    make_agreement(ag, agID, sig_hash, sig, "0000000000000000000000000000000000000000", false)


    // 4. Pass wrong counter party - should not make an agreement, will generate fraud event
    agID = []byte("22222222222222222222222222222222")
    fmt.Printf("Try to make an agreement using the wrong counterParty, ID:%v\n", agID)
    make_agreement(ag, agID, sig_hash, sig, "1111111111111111111111111111111111111111", false)


    // ===================================================================================================
    // Try to terminate something that is not a real agreement.
    //
    log.Printf("Try to fraudulently terminate agreement.\n")
    agID = []byte("22222222222222222222222222222222")
    fmt.Printf("Try to terminate a non-existing agreement, ID:%v\n", agID)
    terminate_agreement(ag, agID, agreements_owner, false)


    // ===================================================================================================
    // Have the admin terminate the first agreement made by this testcase.
    //
    log.Printf("Admin Terminating agreement.\n")

    var res interface{}
    empty_bytes := make([]byte, 32)
    tx_delay_toleration := 120

    p = make([]interface{},0,10)
    p = append(p, agreements_owner)
    p = append(p, agreements_owner)
    agID = []byte("00000000000000000000000000000000")
    p = append(p, agID)
    p = append(p, 20)
    if _, err = ag.Invoke_method("admin_delete_agreement", p); err != nil {
        log.Printf("...terminating, could not invoke terminate_agreement: %v\n", err)
        os.Exit(1)
    }
    log.Printf("Admin Terminate agreement invoked.\n")

    p = make([]interface{},0,10)
    p = append(p, agreements_owner)
    p = append(p, agID)
    start_timer := time.Now()
    for {
        fmt.Printf("There should NOT be a recorded contract hash, but it might still be visible for a few blocks.\n")
        if res, err = ag.Invoke_method("get_contract_hash", p); err == nil {
            fmt.Printf("Received contract hash:%v.\n",res)
            if bytes.Compare([]byte(res.(string)), empty_bytes) != 0 {
                if int(time.Now().Sub(start_timer).Seconds()) < tx_delay_toleration {
                    fmt.Printf("Sleeping, waiting for the block with the Update.\n")
                    time.Sleep(15 * time.Second)
                } else {
                    fmt.Printf("Timeout waiting for the Update.\n")
                    os.Exit(1)
                }
            } else {
                break
            }
        } else {
            fmt.Printf("Error on get_contract_hash: %v\n",err)
            os.Exit(1)
        }
    }

    log.Printf("Admin Terminated agreement.\n")


    // ===================================================================================================
    // Find all events related to the agreement tests in the blockchain and dump them into the output.
    // The events should match the sequence of operations that occurred above.

    log.Printf("Dumping blockchain event data for contract %v.\n",ag.Get_contract_address())
    out, err = "", error(nil)
    rpcResp = new(rpcResponse)
    result := ""

    params := make(map[string]string)
    params["address"] = ag.Get_contract_address()
    params["fromBlock"] = "0x1"

    if out, err = ag.Call_rpc_api("eth_newFilter", params); err == nil {
        if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
            if rpcResp.Error.Message != "" {
                log.Printf("eth_newFilter returned an error: %v.\n", rpcResp.Error.Message)
            } else {
                result = rpcResp.Result.(string)
                // log.Printf("Event id: %v.\n",result)
            }
        }
    }

    rpcFilterResp := new(rpcGetFilterChangesResponse)
    if out, err = ag.Call_rpc_api("eth_getFilterLogs", result); err == nil {
        if err = json.Unmarshal([]byte(out), rpcFilterResp); err == nil {
            if rpcFilterResp.Error.Message != "" {
                log.Printf("eth_getFilterChanges returned an error: %v.\n", rpcFilterResp.Error.Message)
            }
        }
    } else {
        log.Printf("Error calling getFilterLogs: %v.\n",err)
    }

    if len(rpcFilterResp.Result) > 0 {
        for ix, ev := range rpcFilterResp.Result {
            format_ag_event(ix, ev);
        }
    }

    fmt.Println("Terminating agreement protocol test client")
}

// This function is used to invoke the create_agreement function on the blockchain.
// After the invocation is done, it will poll the blockchain to make sure that
// the blockchain was correctly updated. Remember, we run the blockchain such that
// state changes are not visible for 2 or 3 blocks from the head, to avoid reading
// state from forks that end up being thrown away.
func make_agreement(ag *contract_api.SolidityContract, agID []byte, sig_hash string, sig string, counterparty string, shouldWork bool) {
    tx_delay_toleration := 120
    err := error(nil)

    log.Printf("Make an agreement with ID:%v\n", agID)
    p := make([]interface{},0,10)
    p = append(p, agID)
    p = append(p, sig_hash[2:])
    p = append(p, sig[2:])
    p = append(p, counterparty)
    if _, err = ag.Invoke_method("create_agreement", p); err != nil {
        log.Printf("...terminating, could not invoke create_agreement: %v\n", err)
        os.Exit(1)
    }
    log.Printf("Create agreement %v invoked.\n", agID)

    var res interface{}
    p = make([]interface{},0,10)
    p = append(p, counterparty)
    p = append(p, agID)
    byte_hash, _ := hex.DecodeString(sig_hash[2:])
    log.Printf("Binary Hash is: %v\n", byte_hash)
    start_timer := time.Now()
    for {
        fmt.Printf("There should be a recorded contract hash, but it might be in a block we can't read yet.\n")
        if res, err = ag.Invoke_method("get_contract_hash", p); err == nil {
            fmt.Printf("Received contract hash:%v.\n",res)
            if bytes.Compare([]byte(res.(string)), byte_hash) != 0 {
                if int(time.Now().Sub(start_timer).Seconds()) < tx_delay_toleration {
                    fmt.Printf("Sleeping, waiting for the block with the Update.\n")
                    time.Sleep(15 * time.Second)
                } else {
                    if shouldWork {
                        fmt.Printf("Timeout waiting for the Update.\n")
                        os.Exit(1)
                    } else {
                        fmt.Printf("Timeout waiting for the Update. This is expected.\n")
                        break
                    }
                }
            } else {
                if shouldWork {
                    log.Printf("Created agreement %v.\n", agID)
                    break
                } else {
                    fmt.Printf("Received contract hash. This is NOT expected: %v\n", res.(string))
                    os.Exit(2)
                }
            }
        } else {
            fmt.Printf("Error on get_contract_hash: %v\n",err)
            os.Exit(1)
        }
    }
}

// This function is used to invoke the terminate_agreement function on the blockchain.
// After the invocation is done, it will poll the blockchain to make sure that
// the blockchain was correctly updated. Remember, we run the blockchain such that
// state changes are not visible for 2 or 3 blocks from the head, to avoid reading
// state from forks that end up being thrown away.
func terminate_agreement(ag *contract_api.SolidityContract, agID []byte, counterParty string, shouldWork bool) {
    log.Printf("Terminating agreement %v.\n", agID)
    tx_delay_toleration := 120
    err := error(nil)

    p := make([]interface{},0,10)
    p = append(p, counterParty)
    p = append(p, agID)
    p = append(p, 1)
    if _, err = ag.Invoke_method("terminate_agreement", p); err != nil {
        log.Printf("...terminating, could not invoke terminate_agreement: %v\n", err)
        os.Exit(1)
    }
    log.Printf("Terminate agreement %v invoked.\n", agID)

    p = make([]interface{},0,10)
    p = append(p, counterParty)
    p = append(p, agID)
    empty_bytes := make([]byte, 32)
    var res interface{}
    start_timer := time.Now()
    for {
        fmt.Printf("There should NOT be a recorded contract hash, but it might still be visible for a few blocks.\n")
        if res, err = ag.Invoke_method("get_contract_hash", p); err == nil {
            fmt.Printf("Received contract hash:%v.\n",res)
            if shouldWork {
                if bytes.Compare([]byte(res.(string)), empty_bytes) != 0 {
                    if int(time.Now().Sub(start_timer).Seconds()) < tx_delay_toleration {
                        fmt.Printf("Sleeping, waiting for the block with the Update.\n")
                        time.Sleep(15 * time.Second)
                    } else {
                        fmt.Printf("Timeout waiting for the Update.\n")
                        os.Exit(1)
                    }
                } else {
                    log.Printf("Terminated agreement %v.\n", agID)
                    break
                }
            } else {
                if bytes.Compare([]byte(res.(string)), empty_bytes) == 0 {
                    if int(time.Now().Sub(start_timer).Seconds()) < tx_delay_toleration {
                        fmt.Printf("Sleeping, waiting for the block with the Update.\n")
                        time.Sleep(15 * time.Second)
                    } else {
                        fmt.Printf("Timeout waiting for the Update. This is expected\n")
                        break
                    }
                } else {
                    fmt.Printf("Received contract hash. This is NOT expected: %v\n", res.(string))
                    os.Exit(2)
                }
            }
        } else {
            fmt.Printf("Error on get_contract_hash: %v\n",err)
            os.Exit(1)
        }
    }
}

// For an event that is found in the blockchain, format it and write it out to the
// testcase log.
func format_ag_event(ix int, ev rpcFilterChanges) {
    // These event strings correspond to event codes from the agreements contract
    ag_cr8        := "0x0000000000000000000000000000000000000000000000000000000000000000"
    ag_cr8_detail := "0x0000000000000000000000000000000000000000000000000000000000000001"
    ag_cr8_fraud  := "0x0000000000000000000000000000000000000000000000000000000000000002"
    ag_con_term   := "0x0000000000000000000000000000000000000000000000000000000000000003"
    ag_pub_term   := "0x0000000000000000000000000000000000000000000000000000000000000004"
    ag_fraud_term := "0x0000000000000000000000000000000000000000000000000000000000000005"
    ag_adm_term   := "0x0000000000000000000000000000000000000000000000000000000000000006"

    if ev.Topics[0] == ag_cr8 {
        log.Printf("|%03d| Agreement created %v\n",ix,ev.Topics[1]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
    } else if ev.Topics[0] == ag_cr8_detail {
        log.Printf("|%03d| Agreement created detail %v\n",ix,ev.Topics[1]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
    } else if ev.Topics[0] == ag_cr8_fraud {
        log.Printf("|%03d| Agreement creation fraud %v\n",ix,ev.Topics[1]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
    } else if ev.Topics[0] == ag_con_term {
        log.Printf("|%03d| Consumer Terminated %v\n",ix,ev.Topics[1]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
    } else if ev.Topics[0] == ag_pub_term {
        log.Printf("|%03d| Publisher Terminated %v\n",ix,ev.Topics[1]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
    } else if ev.Topics[0] == ag_fraud_term {
        log.Printf("|%03d| Fraudulent Termination %v\n",ix,ev.Topics[1]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
    } else if ev.Topics[0] == ag_adm_term {
        log.Printf("|%03d| Admin Termination %v\n",ix,ev.Topics[1]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
    } else {
        log.Printf("|%03d| Unknown event code in first topic slot.\n")
        log.Printf("Raw log entry:\n%v\n\n",ev)
    }
}

// Mappings of structures used in the ethereum RPC API
type rpcResponse struct {
    Id      string      `json:"id"`
    Version string      `json:"jsonrpc"`
    Result  interface{} `json:"result"`
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

type rpcGetFilterChangesResponse struct {
    Id      string             `json:"id"`
    Version string             `json:"jsonrpc"`
    Result  []rpcFilterChanges `json:"result"`
    Error   struct {
        Code    int    `json:"code"`
        Message string `json:"message"`
    } `json:"error"`
}
