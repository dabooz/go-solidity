package main

import (
    "encoding/json"
    "log"
    "math/rand"
    "os"
    "repo.hovitos.engineering/MTN/go-solidity/contract_api"
    "strings"
    "time"
    )

func main() {
    log.Println("Starting device owner client")

    if len(os.Args) < 3 {
        log.Printf("...terminating, only %v parameters were passed.\n",len(os.Args))
        os.Exit(1)
    }

    err := error(nil)
    rand.Seed(time.Now().UnixNano())

    dir_contract := os.Args[1]
    device_owner := os.Args[2]

    whisper_id := ""
    if len(os.Args) > 3 {
        whisper_id = os.Args[3]
    }
    log.Printf("Using whisper id:%v\n",whisper_id)

    // var new_container_event_code uint64 = 1
    // var execution_complete_event_code uint64 = 2

    // var escrow_cancelled_event_code uint64 = 6
    // var proposer_accepted_event_code uint64 = 8

    // ---------------------- Start of one time init code ------------------------
    //

    // Deploy the device contract
    sc := contract_api.SolidityContractFactory("container_executor")
    if _,err := sc.Deploy_contract(device_owner, ""); err != nil {
        log.Printf("...terminating, could not deploy device contract: %v\n",err)
        os.Exit(1)
    }
    log.Printf("container_executor deployed at %v\n",sc.Get_contract_address())

    // Test to make sure the device contract is invokable
    var owner interface{}
    if owner,err = sc.Invoke_method("get_owner",nil); err != nil {
        log.Printf("...terminating, could not invoke get_owner on device contract: %v\n",err)
        os.Exit(1)
    }
    if owner.(string)[2:] != device_owner {
        log.Printf("...terminating, wrong owner returned: %v should be %v\n",owner,device_owner)
        os.Exit(1)
    }
    log.Printf("Owner is %v\n",owner)

    // Establish the directory contract
    dirc := contract_api.SolidityContractFactory("directory")
    if _,err := dirc.Load_contract(device_owner, ""); err != nil {
        log.Printf("...terminating, could not load directory contract: %v\n",err)
        os.Exit(1)
    }
    dirc.Set_contract_address(dir_contract)

    // Find the token_bank contract
    var tbaddr interface{}
    p := make([]interface{},0,10)
    p = append(p,"token_bank")
    if tbaddr,err = dirc.Invoke_method("get_entry",p); err != nil {
        log.Printf("...terminating, could not find token_bank in directory: %v\n",err)
        os.Exit(1)
    }
    log.Printf("token_bank addr is %v\n",tbaddr)

    // Connect the device contract to the token bank
    p = make([]interface{},0,10)
    p = append(p,tbaddr)
    if _,err := sc.Invoke_method("set_bank",p); err != nil {
        log.Printf("...terminating, could not find token_bank in directory: %v\n",err)
        os.Exit(1)
    }
    log.Printf("Device is connected to token bank\n")

    var echo_bank interface{}
    if echo_bank,err = sc.Invoke_method("get_bank",nil); err != nil {
        log.Printf("...terminating, could not invoke get_bank: %v\n",err)
        os.Exit(1)
    }
    log.Printf("Device using bank at %v.\n",echo_bank)

    // Find the device registry contract
    var draddr interface{}
    p = make([]interface{},0,10)
    p = append(p,"device_registry")
    if draddr,err = dirc.Invoke_method("get_entry",p); err != nil {
        log.Printf("...terminating, could not find device_registry in directory: %v\n",err)
        os.Exit(1)
    }
    log.Printf("device_registry addr is %v\n",draddr)

    // Establish the device_registry contract
    dr := contract_api.SolidityContractFactory("device_registry")
    if _,err := dr.Load_contract(device_owner, ""); err != nil {
        log.Printf("...terminating, could not load device_registry contract: %v\n",err)
        os.Exit(1)
    }
    dr.Set_contract_address(draddr.(string))

    // Register the device in the registry
    p = make([]interface{},0,10)
    p = append(p,sc.Get_contract_address())
    p2 := make([]string,0,20)
    p2 = append(p2,"name")
    p2 = append(p2,"Hello, fib!")
    p2 = append(p2,"arch")
    p2 = append(p2,"armhf")
    p2 = append(p2,"ram")
    p2 = append(p2,"4096")
    p2 = append(p2,"cpus")
    p2 = append(p2,"4")
    p2 = append(p2,"monthly_cap_KB")
    p2 = append(p2,"3278604")
    p2 = append(p2,"hourly_cost_bacon")
    p2 = append(p2,"60")
    p2 = append(p2,"sdr")
    p2 = append(p2,"RTL2832,R820T2")
    p2 = append(p2,"is_seed_enabled")
    p2 = append(p2,"false")
    p2 = append(p2,"is_loc_enabled")
    p2 = append(p2,"true")
    p2 = append(p2,"is_bandwidth_test_enabled")
    p2 = append(p2,"true")
    p = append(p,p2)
    if _,err := dr.Invoke_method("register",p); err != nil {
        log.Printf("...terminating, could not register device: %v\n",err)
        os.Exit(1)
    }

    // Find the device in the registry
    var echo_device interface{}
    p = make([]interface{},0,10)
    p = append(p,sc.Get_contract_address())
    if echo_device,err = dr.Invoke_method("get_description",p); err != nil {
        log.Printf("...terminating, could not invoke get_description: %v\n",err)
        os.Exit(1)
    }
    log.Printf("Device registered with %v.\n",echo_device)

    // Establish the token_bank contract
    bank := contract_api.SolidityContractFactory("token_bank")
    if _,err := bank.Load_contract(device_owner, ""); err != nil {
        log.Printf("...terminating, could not load token_bank contract: %v\n",err)
        os.Exit(1)
    }
    bank.Set_contract_address(tbaddr.(string))

    // Check device bacon balance
    var bal interface{}
    if bal,err = bank.Invoke_method("account_balance",nil); err != nil {
        log.Printf("...terminating, could not get token balance: %v\n",err)
        os.Exit(1)
    }
    log.Printf("Owner bacon balance is:%v\n",bal)

    //
    // ------------------- End of one time initialization ------------------------

    // ------------------- Start of worker loop ----------------------------------
    // First you would use the load_contract API and the set_contract address API
    // to connect to the device contract (container_executor), then the following
    // code.
    //

    for i := 0; i < 5; i++ {

        log.Printf("Waiting for New Container assignment.\n")

        agreement_set := false
        for !agreement_set {
            time.Sleep(5000*time.Millisecond)
            var container_provider interface{}
            if container_provider,err = sc.Invoke_method("get_container_provider",nil); err != nil {
                log.Printf("...terminating, could not get container provider: %v\n",err)
                os.Exit(1)
            }
            if container_provider != "0x0000000000000000000000000000000000000000"  {
                log.Printf("Proposal has been made.\n")
                agreement_set = true
            }
        } // looping for proposal

        var agreement_id interface{}
        if agreement_id,err = sc.Invoke_method("get_agreement_id",nil); err != nil {
            log.Printf("...terminating, could not get agreement id: %v\n",err)
            os.Exit(1)
        }
        log.Printf("Agreement id :%v assigned.\n",agreement_id)

        var whisper interface{}
        if whisper,err = sc.Invoke_method("get_whisper",nil); err != nil {
            log.Printf("...terminating, could not get whisper: %v\n",err)
            os.Exit(1)
        }
        log.Printf("Using whisper:%v.\n",whisper)

        var container_provider interface{}
        if container_provider,err = sc.Invoke_method("get_container_provider",nil); err != nil {
            log.Printf("...terminating, could not get container provider: %v\n",err)
            os.Exit(1)
        }
        log.Printf("Container provider :%v assigned.\n",container_provider)

        var proposal_amount interface{}
        p = make([]interface{},0,10)
        p = append(p,container_provider)
        p = append(p,device_owner)
        p = append(p,sc.Get_contract_address())
        if proposal_amount,err = bank.Invoke_method("get_escrow_amount",p); err != nil {
            log.Printf("...terminating, could not get escrow amount: %v\n",err)
            os.Exit(1)
        }
        log.Printf("Proposal amount: %v\n",proposal_amount)

        cancel := proposal_amount.(uint64)
        if cancel < 3 {
            log.Printf("Deciding to reject this proposal.\n")
            if _,err = sc.Invoke_method("reject_container",nil); err != nil {
                log.Printf("...terminating, could not cancel escrow: %v\n",err)
                os.Exit(1)
            }
        } else {

            p = make([]interface{},0,10)
            p = append(p,container_provider)
            p = append(p,sc.Get_contract_address())
            p = append(p,true)
            if _,err = bank.Invoke_method("counter_party_vote",p); err != nil {
                log.Printf("...terminating, could not send counter party vote: %v\n",err)
                os.Exit(1)
            }

            log.Printf("Pretending to download and run the container.\n")
            time.Sleep(10000*time.Millisecond)

            log.Printf("Waiting for acceptance from proposer.\n")

            found_cancel := false
            agreement_reached := false
            start_timer := time.Now()
            for !agreement_reached && !found_cancel {
                delta := time.Now().Sub(start_timer).Seconds()
                if int(delta) < 150 {
                    time.Sleep(5000*time.Millisecond)
                    var a_reached interface{}
                    p = make([]interface{},0,10)
                    p = append(p,container_provider)
                    p = append(p,device_owner)
                    p = append(p,sc.Get_contract_address())
                    if a_reached,err = bank.Invoke_method("get_proposer_accepted",p); err != nil {
                        log.Printf("...terminating, error checking proposer vote: %v\n",err)
                        os.Exit(1)
                    }
                    agreement_reached = a_reached.(bool)
                    if agreement_reached == true {
                        log.Printf("Governor has accepted.\n")
                    } else {
                        var container_provider interface{}
                        if container_provider,err = sc.Invoke_method("get_container_provider",nil); err != nil {
                            log.Printf("...terminating, could not get container provider: %v\n",err)
                            os.Exit(1)
                        }
                        if container_provider == "0x0000000000000000000000000000000000000000"  {
                            log.Printf("Governor has cancelled instead of accepting.\n")
                            found_cancel = true
                        }
                    }
                } else {
                    log.Printf("Timeout waiting for governor to agree.\n")
                    break
                }
            } // looping for governor acceptance

            if found_cancel == false && agreement_reached == true {
                // stay in the agreement until the governor cancels it
                cancel := false
                for !cancel {
                    time.Sleep(5000*time.Millisecond)
                    var container_provider interface{}
                    if container_provider,err = sc.Invoke_method("get_container_provider",nil); err != nil {
                        log.Printf("...terminating, could not get container provider: %v\n",err)
                        os.Exit(1)
                    }
                    if container_provider == "0x0000000000000000000000000000000000000000" {
                        log.Printf("Governor has cancelled.\n")
                        cancel = true
                    } else {
                        x := rand.Intn(100)
                        if x <= 1 {
                            log.Printf("Device is cancelling.\n")
                            if _,err = sc.Invoke_method("reject_container",nil); err != nil {
                                log.Printf("...terminating, could not reject container: %v\n",err)
                                os.Exit(1)
                            }
                            log.Printf("Contract rejected.\n")
                            cancel = true
                        }
                    }

                } // looping for governor cancel
            } else {
                log.Printf("Resetting device state.\n")
                if _,err = sc.Invoke_method("reject_container",nil); err != nil {
                    log.Printf("...terminating, could not reject container: %v\n",err)
                    os.Exit(1)
                }
                log.Printf("Contract rejected.\n")
            }
        }
        // Check bacon balance
        var bal interface{}
        if bal,err = bank.Invoke_method("account_balance",nil); err != nil {
            log.Printf("...terminating, could not get token balance: %v\n",err)
            os.Exit(1)
        }
        log.Printf("Owner bacon balance is:%v\n",bal)

    }

    log.Printf("Deregistering the device\n")
    p = make([]interface{},0,10)
    p = append(p,sc.Get_contract_address())
    if _,err := dr.Invoke_method("deregister",p); err != nil {
        log.Printf("...terminating, could not deregister device: %v\n",err)
        os.Exit(1)
    }

    // Find all events related to this test in the blockchain and dump them into the output.

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

    // These event string correspond to event codes from the container_executor contract
    dev_ev_prop := "0x0000000000000000000000000000000000000000000000000000000000000000"
    dev_ev_perr := "0x0000000000000000000000000000000000000000000000000000000000000001"
    dev_ev_rej  := "0x0000000000000000000000000000000000000000000000000000000000000002"
    dev_ev_can  := "0x0000000000000000000000000000000000000000000000000000000000000003"

    log.Printf("Dumping blockchain event data for contract %v.\n",sc.Get_contract_address())
    result, out, err := "", "", error(nil)
    var rpcResp *rpcResponse = new(rpcResponse)

    params := make(map[string]string)
    params["address"] = sc.Get_contract_address()
    params["fromBlock"] = "0x1"

    if out, err = sc.Call_rpc_api("eth_newFilter", params); err == nil {
        if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
            if rpcResp.Error.Message != "" {
                log.Printf("eth_newFilter returned an error: %v.\n", rpcResp.Error.Message)
            } else {
                result = rpcResp.Result.(string)
                // log.Printf("Event id: %v.\n",result)
            }
        }
    }

    var rpcFilterResp *rpcGetFilterChangesResponse = new(rpcGetFilterChangesResponse)
    if out, err = sc.Call_rpc_api("eth_getFilterLogs", result); err == nil {
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
            if ev.Topics[0] == dev_ev_prop {
                log.Printf("|%03d| New Proposal from %v\n",ix,ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == dev_ev_perr {
                log.Printf("|%03d| New Proposal Error from %v\n",ix,ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == dev_ev_rej {
                log.Printf("|%03d| Proposal rejected by %v\n",ix,ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == dev_ev_can {
                log.Printf("|%03d| Proposal cancelled by %v\n",ix,ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else {
                log.Printf("|%03d| Unknown event code in first topic slot.\n")
                log.Printf("Raw log entry:\n%v\n\n",ev)
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            }
        }
    }

    // These event string correspond to event codes from the token_bank contract

    b_ev_mint                           := "0x0000000000000000000000000000000000000000000000000000000000000000"
    b_ev_loan_created                   := "0x0000000000000000000000000000000000000000000000000000000000000001"
    b_ev_loan_extended                  := "0x0000000000000000000000000000000000000000000000000000000000000002"
    b_ev_loan_repaid                    := "0x0000000000000000000000000000000000000000000000000000000000000003"
    b_ev_transfer                       := "0x0000000000000000000000000000000000000000000000000000000000000004"
    b_ev_escrow_created                 := "0x0000000000000000000000000000000000000000000000000000000000000005"
    b_ev_escrow_cancelled               := "0x0000000000000000000000000000000000000000000000000000000000000006"
    b_ev_escrow_counterparty_accepted   := "0x0000000000000000000000000000000000000000000000000000000000000007"
    b_ev_escrow_proposer_accepted       := "0x0000000000000000000000000000000000000000000000000000000000000008"
    b_ev_escrow_proposer_paid           := "0x0000000000000000000000000000000000000000000000000000000000000009"
    b_ev_escrow_refunded                := "0x000000000000000000000000000000000000000000000000000000000000000a"


    log.Printf("Dumping blockchain event data for bank transactions involving this owner %v as device owner.\n",device_owner)

    fparams := make(map[string]interface{})
    fparams["address"] = bank.Get_contract_address()
    topics := make([]interface{},0,10)
    topics = append(topics, nil)
    topics = append(topics, nil)
    topics = append(topics,"0x"+strings.Repeat("0", (64-len(device_owner)))+device_owner)
    fparams["topics"] = topics
    fparams["fromBlock"] = "0x1"

    if out, err = sc.Call_rpc_api("eth_newFilter", fparams); err == nil {
        if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
            if rpcResp.Error.Message != "" {
                log.Printf("eth_newFilter returned an error: %v.\n", rpcResp.Error.Message)
            } else {
                result = rpcResp.Result.(string)
                // log.Printf("Event id: %v.\n",result)
            }
        }
    }

    if out, err = sc.Call_rpc_api("eth_getFilterLogs", result); err == nil {
        if err = json.Unmarshal([]byte(out), rpcFilterResp); err == nil {
            if rpcFilterResp.Error.Message != "" {
                log.Printf("eth_getFilterLogs returned an error: %v.\n", rpcFilterResp.Error.Message)
            }
        }
    } else {
        log.Printf("Error calling getFilterLogs: %v.\n",err)
    }

    if len(rpcFilterResp.Result) > 0 {
        for ix, ev := range rpcFilterResp.Result {
            if ev.Topics[0] == b_ev_mint {
                log.Printf("|%03d| Mint %v tokens for %v\n",ix,ev.Topics[3],ev.Topics[2]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == b_ev_loan_created {
                log.Printf("|%03d| New Loan %v tokens for %v\n",ix,ev.Topics[2],ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == b_ev_loan_extended {
                log.Printf("|%03d| Loan increased %v tokens for %v\n",ix,ev.Topics[2],ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == b_ev_loan_repaid {
                log.Printf("|%03d| Loan repaid %v tokens for %v\n",ix,ev.Topics[2],ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == b_ev_transfer {
                log.Printf("|%03d| Transfer %v tokens from %v to %v\n",ix,ev.Topics[3],ev.Topics[1],ev.Topics[2]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == b_ev_escrow_created {
                log.Printf("|%03d| Create Escrow %v tokens by %v\n",ix,ev.Data,ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == b_ev_escrow_cancelled {
                log.Printf("|%03d| Cancel Escrow\n",ix);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == b_ev_escrow_counterparty_accepted {
                log.Printf("|%03d| CounterParty Acceptance with %v\n",ix,ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == b_ev_escrow_proposer_accepted {
                log.Printf("|%03d| Proposer Acceptance from %v\n",ix,ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == b_ev_escrow_proposer_paid {
                log.Printf("|%03d| Received %v tokens from %v\n",ix,ev.Data,ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == b_ev_escrow_refunded {
                log.Printf("|%03d| Refunded %v escrowed tokens to %v\n",ix,ev.Data,ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else {
                log.Printf("|%03d| Unknown event code in first topic slot.\n")
                log.Printf("Raw log entry:\n%v\n",ev)
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            }
        }
    }

    // These event string correspond to event codes from the device_registry contract

    dr_ev_new       := "0x0000000000000000000000000000000000000000000000000000000000000000"
    dr_ev_update    := "0x0000000000000000000000000000000000000000000000000000000000000001"
    dr_ev_dereg     := "0x0000000000000000000000000000000000000000000000000000000000000002"

    log.Printf("Dumping blockchain event data for device registry transactions involving this owner %v.\n",device_owner)

    fparams = make(map[string]interface{})
    fparams["address"] = dr.Get_contract_address()
    topics = make([]interface{},0,10)
    topics = append(topics, nil)
    topics = append(topics, nil)
    topics = append(topics,"0x"+strings.Repeat("0", (64-len(device_owner)))+device_owner)
    fparams["topics"] = topics
    fparams["fromBlock"] = "0x1"

    if out, err = dr.Call_rpc_api("eth_newFilter", fparams); err == nil {
        if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
            if rpcResp.Error.Message != "" {
                log.Printf("eth_newFilter returned an error: %v.\n", rpcResp.Error.Message)
            } else {
                result = rpcResp.Result.(string)
                // log.Printf("Event id: %v.\n",result)
            }
        }
    }

    if out, err = dr.Call_rpc_api("eth_getFilterLogs", result); err == nil {
        if err = json.Unmarshal([]byte(out), rpcFilterResp); err == nil {
            if rpcFilterResp.Error.Message != "" {
                log.Printf("eth_getFilterLogs returned an error: %v.\n", rpcFilterResp.Error.Message)
            }
        }
    } else {
        log.Printf("Error calling getFilterLogs: %v.\n",err)
    }

    if len(rpcFilterResp.Result) > 0 {
        for ix, ev := range rpcFilterResp.Result {
            if ev.Topics[0] == dr_ev_new {
                log.Printf("|%03d| New registration of %v\n",ix,ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == dr_ev_update {
                log.Printf("|%03d| Update registration of %v\n",ix,ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else if ev.Topics[0] == dr_ev_dereg {
                log.Printf("|%03d| Deregister %v\n",ix,ev.Topics[1]);
                log.Printf("Data: %v\n",ev.Data);
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            } else {
                log.Printf("|%03d| Unknown event code in first topic slot.\n")
                log.Printf("Raw log entry:\n%v\n",ev)
                log.Printf("Block: %v\n\n",ev.BlockNumber);
            }
        }
    }

    log.Println("Terminating client")
}

