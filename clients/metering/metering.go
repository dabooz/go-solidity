package main

import (
    //"bytes"
    "fmt"
    "encoding/hex"
    "encoding/json"
    "github.com/open-horizon/go-solidity/contract_api"
    "golang.org/x/crypto/sha3"
    "log"
    "math/big"
    "math/rand"
    "os"
    "strings"
    "time"
    )

func main() {
    fmt.Println("Starting metering client")

    if len(os.Args) < 3 {
        fmt.Printf("...terminating, only %v parameters were passed.\n",len(os.Args))
        os.Exit(1)
    }

    tr := new(TestResults)
    numSuccess := 0
    numFraud := 0
    numDelete := 0

    dir_contract := os.Args[1]
    if !strings.HasPrefix(dir_contract, "0x") {
        dir_contract = "0x" + dir_contract
    }
    fmt.Printf("using directory %v\n",dir_contract)
    owner := os.Args[2]
    if !strings.HasPrefix(owner, "0x") {
        owner = "0x" + owner
    }
    fmt.Printf("using account %v\n",owner)

    err := error(nil)

    // Establish the directory contract
    dirc := contract_api.SolidityContractFactory("directory")
    if _,err := dirc.Load_contract(owner, ""); err != nil {
        fmt.Printf("...terminating, could not load directory contract: %v\n",err)
        os.Exit(1)
    }
    dirc.Set_contract_address(dir_contract)

    // Find the metering contract
    var agaddr interface{}
    p := make([]interface{},0,10)
    p = append(p,"metering")
    if agaddr,err = dirc.Invoke_method("get_entry",p); err != nil {
        log.Printf("...terminating, could not find metering in directory: %v\n",err)
        os.Exit(1)
    }
    log.Printf("metering addr is %v\n",agaddr)

    // Establish the metering contract
    ag := contract_api.SolidityContractFactory("metering")
    if _,err := ag.Load_contract(owner, ""); err != nil {
        log.Printf("...terminating, could not load metering contract: %v\n",err)
        os.Exit(1)
    }
    ag.Set_contract_address(agaddr.(string))


    // ===================================================================================================
    // Prepare to make the first create_meter call with a simple set of parameters and leave it in
    // the system. Here we get a hash of a test string and then sign the string. These are used throughout
    // all the tests.
    //
    log.Printf("Hash and sign a simple string.\n")
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

    if out, err = ag.Call_rpc_api("eth_sign", contract_api.MultiValueParams{owner, sig_hash}); err == nil {
        if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
            if rpcResp.Error.Message != "" {
                log.Printf("RPC sign of terms and conditions hash failed, error: %v.", rpcResp.Error.Message)
                os.Exit(1)
            } else {
                sig = rpcResp.Result.(string)
                log.Printf("Signature of terms and conditions hash is: %v\n", sig)
            }
        } else {
            log.Printf("Unmarshal error: %v.", err)
            os.Exit(1)
        }
    }

    if len(sig[2:]) != 130 {
        log.Printf("Signature has wrong length: %v.", len(sig[2:]))
        os.Exit(1)
    }

    // ===================================================================================================
    // This is the heart of the testcase. Here we will start trying to make meters on the blockchain.
    //

    // ===================================================================================================
    // Loop creating meters, pushing ethereum to see if it drops any pending transactions.  The loop
    // count can be pushed up to create more stress.

    maxLoops := 10
    random := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
    agId := generateAgreementId(random)
    inc := big.NewInt(1)
    newId := big.NewInt(0)

    for x := 1; x <= maxLoops; x++ {
        log.Printf("Make a simple meter %v in a loop using %v\n", x, agId)
        currentTime := time.Now().Unix()
        meterHash := getMeterHash(uint64(x+100), uint64(currentTime), agId)
        meterSig := getMeterSig(ag, meterHash, owner)
        only_make_meter(ag, uint64(x+100), uint64(currentTime), agId, meterHash, meterSig, sig_hash, sig, owner)

        newId = newId.SetBytes(agId)
        newId = newId.Add(newId, inc)
        agId = newId.Bytes()

    }

    log.Printf("Done looping test.\n")
    numSuccess += maxLoops

    // Invoke the blockchain to make the first meter.
    //
    agID := []byte("00000000000000000000000000000000")
    count := 1
    currentTime := time.Now().Unix()
    log.Printf("Make a simple meter using %v\n", agID)
    make_meter(ag, uint64(count), uint64(currentTime), agID, sig_hash, sig, owner, true)
    numSuccess += 1

    // ===================================================================================================
    // Make another create_meter call and then delete it.
    //
    agID = []byte("11111111111111111111111111111111")
    fmt.Printf("Make a second meter using ID: %v\n", agID)
    count = 2
    currentTime = time.Now().Unix()
    make_meter(ag, uint64(count), uint64(currentTime), agID, sig_hash, sig, owner, true)
    numSuccess += 1

    terminate_meter(ag, agID, owner, true)
    numDelete += 1


    // ===================================================================================================
    // Try to make some meters with incorrect info to prove that the smart contract will reject
    // invalid meter attempts or invalid terminations.
    //

    // 1. Use an invalid signature - should not make a meter, will generate fraud event
    agID = []byte("22222222222222222222222222222222")
    fmt.Printf("Try to make a meter using an invalid signature, ID:%v\n", agID)
    count = 3
    currentTime = time.Now().Unix()
    make_meter(ag, uint64(count), uint64(currentTime), agID, sig_hash, "012345678901234567890123456789012345678901234567890123456789", owner, false)
    numFraud += 1

    // 2. Use an existing agreement ID with a different hash - should not make a meter, 
    agID = []byte("00000000000000000000000000000000")
    fmt.Printf("Try to make a meter using an invalid hash, ID:%v\n", agID)
    count = 4
    currentTime = time.Now().Unix()
    make_meter(ag, uint64(count), uint64(currentTime), agID, "11111111111111111111111111111111", sig, owner, false)
    numFraud += 1

    // 3. Pass no counter party - should not make a meter
    agID = []byte("22222222222222222222222222222222")
    fmt.Printf("Try to make a meter without providing a counterParty, ID:%v\n", agID)
    count = 5
    currentTime = time.Now().Unix()
    meterHash := getMeterHash(uint64(count), uint64(currentTime), agID)
    meterSig := getMeterSig(ag, meterHash, owner)
    make_meter_long(ag, uint64(count), uint64(currentTime), agID, meterHash, meterSig, sig_hash, sig, "0x0000000000000000000000000000000000000000", false)

    // 4. Pass wrong counter party - should not make a meter, will generate fraud event
    agID = []byte("22222222222222222222222222222222")
    fmt.Printf("Try to make a meter using the wrong counterParty, ID:%v\n", agID)
    count = 6
    currentTime = time.Now().Unix()
    meterHash = getMeterHash(uint64(count), uint64(currentTime), agID)
    meterSig = getMeterSig(ag, meterHash, owner)
    make_meter_long(ag, uint64(count), uint64(currentTime), agID, meterHash, meterSig, sig_hash, sig, "0x1111111111111111111111111111111111111111", false)
    numFraud += 1

    // 5. Pass zero count - should not make a meter
    agID = []byte("22222222222222222222222222222222")
    fmt.Printf("Try to make a meter using a zero count, ID:%v\n", agID)
    count = 7
    currentTime = time.Now().Unix()
    make_meter(ag, uint64(0), uint64(currentTime), agID, sig_hash, sig, owner, false)

    // 6. Pass zero time - should not make a meter
    agID = []byte("22222222222222222222222222222222")
    fmt.Printf("Try to make a meter using a zero time, ID:%v\n", agID)
    count = 8
    currentTime = time.Now().Unix()
    make_meter(ag, uint64(count), uint64(0), agID, sig_hash, sig, owner, false)

    // 7. Pass invalid meter hash, will generate fraud event
    agID = []byte("22222222222222222222222222222222")
    fmt.Printf("Try to make a meter using an invalid meter hash, ID:%v\n", agID)
    count = 9
    currentTime = time.Now().Unix()
    meterHash = getMeterHash(uint64(count), uint64(currentTime), agID)
    meterSig = getMeterSig(ag, meterHash, owner)
    badHash := "0x1111000000000011111111112222222222333333333344444444445555555555"
    make_meter_long(ag, uint64(count), uint64(currentTime), agID, badHash, meterSig, sig_hash, sig, owner, false)
    numFraud += 1

    // 8. Pass invalid meter hash signature, will generate fraud event
    agID = []byte("22222222222222222222222222222222")
    fmt.Printf("Try to make a meter using an invalid meter hash signature, ID:%v\n", agID)
    count = 10
    currentTime = time.Now().Unix()
    meterHash = getMeterHash(uint64(count), uint64(currentTime), agID)
    meterSig = "0x1111"
    make_meter_long(ag, uint64(count), uint64(currentTime), agID, meterHash, meterSig, sig_hash, sig, owner, false)
    numFraud += 1

    // ===================================================================================================
    // Try to terminate something that is not a real agreement.
    //
    log.Printf("Try to fraudulently terminate agreement.\n")
    agID = []byte("22222222222222222222222222222222")
    fmt.Printf("Try to terminate a non-existing agreement, ID:%v\n", agID)
    terminate_meter(ag, agID, owner, true)

    // ===================================================================================================
    // Have the admin terminate the first agreement made by this testcase.
    //
    agID = []byte("00000000000000000000000000000000")
    terminate_meter(ag, agID, owner, true)
    numDelete += 1

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
            format_m_event(ix, ev, tr);
        }
    }

    // Verify the results
    if tr.Successful != numSuccess || tr.Fraud != numFraud || tr.Delete != numDelete {
        log.Printf("Error checking test results: %v.\n", tr)
        log.Printf("Expected Success: %v, Fraud: %v, Delete: %v", numSuccess, numFraud, numDelete)
        os.Exit(1)
    } else {
        log.Printf("Test results all pass.\n")
    }

    fmt.Println("Terminating metering test client")
}

// This function is used to invoke the create_meter function on the blockchain.
// After the invocation is done, it will poll the blockchain to make sure that
// the blockchain was correctly updated. Remember, we run the blockchain such that
// state changes are not visible for 2 or 3 blocks from the head, to avoid reading
// state from forks that end up being thrown away.
func make_meter(ag *contract_api.SolidityContract, count uint64, currentTime uint64, agID []byte, sig_hash string, sig string, counterparty string, shouldWork bool) {

    meterHash := getMeterHash(count, currentTime, agID)
    meterSig := getMeterSig(ag, meterHash, counterparty)
    make_meter_long(ag, count, currentTime, agID, meterHash, meterSig, sig_hash, sig, counterparty, shouldWork)

}

func make_meter_long(ag *contract_api.SolidityContract, count uint64, currentTime uint64, agID []byte, meterHash string, meterSig string, sig_hash string, sig string, counterparty string, shouldWork bool) {
    tx_delay_toleration := 120
    err := error(nil)
    var meterCount uint64
    var meterTime uint64

    only_make_meter(ag, count, currentTime, agID, meterHash, meterSig, sig_hash, sig, counterparty)

    var res interface{}
    p := make([]interface{},0,10)
    p = append(p, agID)
    p = append(p, counterparty)

    start_timer := time.Now()
    for {
        if shouldWork {
            fmt.Printf("There should be a recorded meter, but it might be in a block we can't read yet.\n")
        } else {
            fmt.Printf("There should NOT be a recorded meter.\n")
        }
        if res, err = ag.Invoke_method("read_meter", p); err == nil {
            fmt.Printf("Received meter:%T %v.\n",res, res)
            switch res.(type) {
            case []interface{}:
                resultArray := res.([]interface{})
                if len(resultArray) != 2 {
                    fmt.Printf("Return value does not have enough array elements: %v, should be 2.\n", len(resultArray))
                    os.Exit(1)
                }
                switch resultArray[0].(type) {
                case uint64:
                    meterCount = resultArray[0].(uint64)
                default:
                    fmt.Printf("Wrong return type from invocation: %T, should be uint64.\n", resultArray[0])
                    os.Exit(1)
                }
                switch resultArray[1].(type) {
                case uint64:
                    meterTime = resultArray[1].(uint64)
                default:
                    fmt.Printf("Wrong return type from invocation: %T, should be uint64.\n", resultArray[0])
                    os.Exit(1)
                }
            default:
                fmt.Printf("Wrong return type from invocation, should be array.\n")
                os.Exit(1)
            }

            if meterCount != count || meterTime != currentTime {
                if int(time.Now().Sub(start_timer).Seconds()) < tx_delay_toleration {
                    fmt.Printf("Sleeping, waiting for the block with the Update.\n")
                    time.Sleep(15 * time.Second)
                } else {
                    if shouldWork {
                        fmt.Printf("Timeout waiting for the Update. This is NOT expected.\n")
                        os.Exit(1)
                    } else {
                        fmt.Printf("Timeout waiting for the Update. This is expected.\n")
                        break
                    }
                }
            } else {
                if shouldWork {
                    log.Printf("Success: Created meter %v.\n", agID)
                    break
                } else {
                    fmt.Printf("Received meter. This is NOT expected: %v\n", res.(string))
                    os.Exit(2)
                }
            }
        } else {
            fmt.Printf("Error on read_meter: %v\n",err)
            os.Exit(1)
        }
    }
}

// This function is used to write a meter on the blockchain, but it doesnt wait for the write to be visible.
func only_make_meter(ag *contract_api.SolidityContract, count uint64, currentTime uint64, agID []byte, meterHash string, meterSig string, sig_hash string, sig string, counterparty string) {
    err := error(nil)

    log.Printf("Make a meter with ID:%v\n", agID)
    p := make([]interface{},0,10)
    p = append(p, count)
    p = append(p, currentTime)
    p = append(p, agID)
    p = append(p, meterHash[2:])
    p = append(p, meterSig[2:])
    p = append(p, sig_hash[2:])
    p = append(p, sig[2:])
    p = append(p, sig[2:])
    p = append(p, counterparty)
    if _, err = ag.Invoke_method("create_meter", p); err != nil {
        log.Printf("...terminating, could not invoke create_meter: %v\n", err)
        os.Exit(1)
    }
    log.Printf("Create meter %v successfully submitted.\n", agID)
}

// This function is used to invoke the admin_delete_meter function on the blockchain.
// After the invocation is done, it will poll the blockchain to make sure that
// the blockchain was correctly updated. Remember, we run the blockchain such that
// state changes are not visible for 2 or 3 blocks from the head, to avoid reading
// state from forks that end up being thrown away.
func terminate_meter(ag *contract_api.SolidityContract, agID []byte, counterParty string, shouldWork bool) {
    log.Printf("Terminating agreement %v.\n", agID)
    tx_delay_toleration := 120
    err := error(nil)
    var meterCount uint64
    var meterTime uint64

    p := make([]interface{},0,10)
    p = append(p, counterParty)
    p = append(p, counterParty)
    p = append(p, agID)
    if _, err = ag.Invoke_method("admin_delete_meter", p); err != nil {
        log.Printf("...terminating, could not invoke admin_delete_meter: %v\n", err)
        os.Exit(1)
    }
    log.Printf("Admin Delete Meter %v invoked.\n", agID)

    p = make([]interface{},0,10)
    p = append(p, agID)
    p = append(p, counterParty)
    var res interface{}
    start_timer := time.Now()
    for {
        fmt.Printf("There should NOT be a recorded meter, but it might still be visible for a few blocks.\n")
        if res, err = ag.Invoke_method("read_meter", p); err == nil {
            fmt.Printf("Received meter:%v.\n",res)
            switch res.(type) {
            case []interface{}:
                resultArray := res.([]interface{})
                if len(resultArray) != 2 {
                    fmt.Printf("Return value does not have enough array elements: %v, should be 2.\n", len(resultArray))
                    os.Exit(1)
                }
                switch resultArray[0].(type) {
                case uint64:
                    meterCount = resultArray[0].(uint64)
                default:
                    fmt.Printf("Wrong return type from invocation: %T, should be uint64.\n", resultArray[0])
                    os.Exit(1)
                }
                switch resultArray[1].(type) {
                case uint64:
                    meterTime = resultArray[1].(uint64)
                default:
                    fmt.Printf("Wrong return type from invocation: %T, should be uint64.\n", resultArray[0])
                    os.Exit(1)
                }
            default:
                fmt.Printf("Wrong return type from invocation, should be array.\n")
                os.Exit(1)
            }

            if shouldWork {
                if meterCount != 0 || meterTime != 0 {
                    if int(time.Now().Sub(start_timer).Seconds()) < tx_delay_toleration {
                        fmt.Printf("Sleeping, waiting for the block with the Update.\n")
                        time.Sleep(15 * time.Second)
                    } else {
                        fmt.Printf("Timeout waiting for the Delete. This is NOT expected.\n")
                        os.Exit(1)
                    }
                } else {
                    log.Printf("Deleted meter for %v.\n", agID)
                    break
                }
            } else {
                if meterCount != 0 && meterTime != 0  {
                    if int(time.Now().Sub(start_timer).Seconds()) < tx_delay_toleration {
                        fmt.Printf("Sleeping, waiting for the block with the Update.\n")
                        time.Sleep(15 * time.Second)
                    } else {
                        fmt.Printf("Timeout waiting for the Delete. This is expected\n")
                        break
                    }
                } else {
                    fmt.Printf("Meter was deleted. This is NOT expected: %v\n", res)
                    os.Exit(2)
                }
            }
        } else {
            fmt.Printf("Error on read_meter: %v\n",err)
            os.Exit(1)
        }
    }
}

// For an event that is found in the blockchain, format it and write it out to the
// testcase log.
func format_m_event(ix int, ev rpcFilterChanges, tr *TestResults) {
    // These event strings correspond to event codes from the agreements contract
    m_cr8        := "0x0000000000000000000000000000000000000000000000000000000000000000"
    m_cr8_detail := "0x0000000000000000000000000000000000000000000000000000000000000001"
    m_cr8_fraud  := "0x0000000000000000000000000000000000000000000000000000000000000002"
    m_adm_term   := "0x0000000000000000000000000000000000000000000000000000000000000003"
    m_debug      := "0x0000000000000000000000000000000000000000000000000000000000000004"

    if ev.Topics[0] == m_cr8 {
        log.Printf("|%03d| Meter created %v\n",ix,ev.Topics[3]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
        tr.Successful += 1
    } else if ev.Topics[0] == m_cr8_detail {
        log.Printf("|%03d| Meter created detail %v\n",ix,ev.Topics[3]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
    } else if ev.Topics[0] == m_cr8_fraud {
        log.Printf("|%03d| Meter creation fraud %v\n",ix,ev.Topics[3]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
        tr.Fraud += 1
    } else if ev.Topics[0] == m_adm_term {
        log.Printf("|%03d| Admin Deletion %v\n",ix,ev.Topics[3]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
        tr.Delete += 1
    } else if ev.Topics[0] == m_debug {
        log.Printf("|%03d| Debug %v\n",ix,ev.Topics);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
    } else {
        log.Printf("|%03d| Unknown event code in first topic slot.\n")
        log.Printf("Raw log entry:\n%v\n\n",ev)
    }
}

// Testcase result tracking
type TestResults struct {
    Successful int
    Fraud      int
    Delete     int
}

func (t *TestResults) String() string {
    return fmt.Sprintf("Success: %v, Fraud: %v, Delete: %v", t.Successful, t.Fraud, t.Delete)
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

func generateAgreementId(random *rand.Rand) []byte {

    b := make([]byte, 32, 32)
    for i := range b {
        b[i] = byte(random.Intn(256))
    }
    return b
}

func getMeterHash(count uint64, time uint64, agid []byte) string {

    theMeter := make([]byte, 0, 96)
    theMeter = append(theMeter, toBuffer(count)...)
    theMeter = append(theMeter, toBuffer(time)...)
    theMeter = append(theMeter, agid...)

    hash := sha3.Sum256(theMeter)
    hash_string := "0x" + hex.EncodeToString(hash[:])

    return hash_string
}

func getMeterSig(msc *contract_api.SolidityContract, meterHash string, owner string) string {

    var rpcResp *rpcResponse = new(rpcResponse)

    if out, err := msc.Call_rpc_api("eth_sign", contract_api.MultiValueParams{owner, meterHash}); err == nil {
        if err := json.Unmarshal([]byte(out), rpcResp); err == nil {
            if rpcResp.Error.Message != "" {
                log.Printf("RPC sign failed, error: %v.", rpcResp.Error.Message)
                os.Exit(1)
            } else {
                sig := rpcResp.Result.(string)
                log.Printf("Signature of hash is: %v\n", sig)
                return sig
            }
        } else {
            log.Printf("Unmarshal error: %v.", err)
            os.Exit(1)
        }
    } else {
        log.Printf("RPC sign failed, error: %v.", err)
    }

    return ""

}

func toBuffer(seq uint64) []byte {
    buf := make([]byte, 32)
    for i := len(buf) - 1; seq != 0; i-- {
        buf[i] = byte(seq & 0xff)
        seq >>= 8
    }
    return buf
}
