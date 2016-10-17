package main

import (
    "fmt"
    "encoding/json"
    "log"
    "github.com/open-horizon/go-solidity/contract_api"
    "os"
    "strings"
    "time"
    )

func main() {
    fmt.Println("Starting directory client")

    tx_delay_toleration := 180

    if len(os.Args) < 4 {
        fmt.Printf("...terminating, only %v parameters were passed.\n",len(os.Args))
        os.Exit(1)
    }

    dir_contract := os.Args[1]
    fmt.Printf("using directory %v\n",dir_contract)
    registry_owner := os.Args[2]
    fmt.Printf("using account %v\n",registry_owner)

    // Establish the directory contract
    dirc := contract_api.SolidityContractFactory("directory")
    if _,err := dirc.Load_contract(registry_owner, ""); err != nil {
        fmt.Printf("...terminating, could not load directory contract: %v\n",err)
        os.Exit(1)
    }
    dirc.Set_contract_address(dir_contract)

    // Test to make sure the directory contract is invokable
    fmt.Printf("Retrieve contract for name 'a', should be zeroes.\n")
    p := make([]interface{},0,10)
    p = append(p,"a")
    if caddr,err := dirc.Invoke_method("get_entry",p); err == nil {
        fmt.Printf("Contract Address is %v\n",caddr)
        if caddr.(string) != "0x0000000000000000000000000000000000000000" {
            os.Exit(1)
        }
    } else {
        fmt.Printf("Error invoking get_entry: %v\n",err)
        os.Exit(1)
    }

    // fmt.Printf("Retrieve a list of all registered names, should have only the MTN platform entries.\n")
    // p = make([]interface{},0,10)
    // p = append(p,0)
    // p = append(p,10)
    // if nl,err := dirc.Invoke_method("get_names",p); err == nil {
    //     fmt.Printf("Registered names %v\n",nl)
    // } else {
    //     fmt.Printf("Error invoking get_names: %v\n",err)
    //     os.Exit(1)
    // }

    fmt.Printf("Register 'a' with address 0x0000000000000000000000000000000000000010.\n")
    p = make([]interface{},0,10)
    p = append(p,"a")
    p = append(p,"0x0000000000000000000000000000000000000010")
    p = append(p,0)
    if _,err := dirc.Invoke_method("add_entry",p); err == nil {
        fmt.Printf("Registered 'a'.\n")
    } else {
        fmt.Printf("Error invoking add_entry: %v\n",err)
        os.Exit(1)
    }

    start_timer := time.Now()
    for {
        fmt.Printf("Retrieve 'a', should have address 10.\n")
        p = make([]interface{},0,10)
        p = append(p,"a")
        if aa,err := dirc.Invoke_method("get_entry",p); err == nil {
            fmt.Printf("Retrieved 'a', is %v.\n",aa)
            if aa.(string) != "0x0000000000000000000000000000000000000010" {
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
            fmt.Printf("Error invoking add_entry: %v\n",err)
            os.Exit(1)
        }
    }

    // fmt.Printf("Retrieve owner of 'a', should be %v.\n",registry_owner)
    // p = make([]interface{},0,10)
    // p = append(p,"a")
    // p = append(p,0)
    // if aa,err := dirc.Invoke_method("get_entry_owner",p); err == nil {
    //     fmt.Printf("Retrieved owner of 'a' %v.\n",aa)
    //     if aa.(string)[2:] != registry_owner {
    //         os.Exit(1)
    //     }
    // } else {
    //     fmt.Printf("Error invoking add_entry: %v\n",err)
    //     os.Exit(1)
    // }

    // fmt.Printf("Retrieve a list of all registered names, should have 'a' in it.\n")
    // p = make([]interface{},0,10)
    // p = append(p,0)
    // p = append(p,10)
    // if nl,err := dirc.Invoke_method("get_names",p); err == nil {
    //     fmt.Printf("Registered names %v\n",nl)
    // } else {
    //     fmt.Printf("Error invoking get_names: %v\n",err)
    //     os.Exit(1)
    // }

    fmt.Printf("Register 'b' with address 0x0000000000000000000000000000000000000011.\n")
    p = make([]interface{},0,11)
    p = append(p,"b")
    p = append(p,"0x0000000000000000000000000000000000000011")
    p = append(p,0)
    if _,err := dirc.Invoke_method("add_entry",p); err == nil {
        fmt.Printf("Registered 'b'.\n")
    } else {
        fmt.Printf("Error invoking add_entry: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Register 'c' with address 0x0000000000000000000000000000000000000012.\n")
    p = make([]interface{},0,11)
    p = append(p,"c")
    p = append(p,"0x0000000000000000000000000000000000000012")
    p = append(p,0)
    if _,err := dirc.Invoke_method("add_entry",p); err == nil {
        fmt.Printf("Registered 'c'.\n")
    } else {
        fmt.Printf("Error invoking add_entry: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Register 'c' with address 0x0000000000000000000000000000000000000013, version 1.\n")
    p = make([]interface{},0,11)
    p = append(p,"c")
    p = append(p,"0x0000000000000000000000000000000000000013")
    p = append(p,1)
    if _,err := dirc.Invoke_method("add_entry",p); err == nil {
        fmt.Printf("Registered 'c, version 1'.\n")
    } else {
        fmt.Printf("Error invoking add_entry: %v\n",err)
        os.Exit(1)
    }

    // fmt.Printf("Retrieve a list of all registered names, should have 'a,b,c,c' in it.\n")
    // p = make([]interface{},0,10)
    // p = append(p,0)
    // p = append(p,10)
    // if nl,err := dirc.Invoke_method("get_names",p); err == nil {
    //     fmt.Printf("Registered names %v\n",nl)
    // } else {
    //     fmt.Printf("Error invoking get_names: %v\n",err)
    //     os.Exit(1)
    // }

    start_timer = time.Now()
    for {
        fmt.Printf("Retrieve verison 1 of 'c',should be 0x00..0013.\n")
        p = make([]interface{},0,10)
        p = append(p,"c")
        p = append(p,1)
        if nl,err := dirc.Invoke_method("get_entry_by_version",p); err == nil {
            fmt.Printf("Registered c version 1 as %v\n",nl)
            if nl.(string) != "0x0000000000000000000000000000000000000013" {
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
            fmt.Printf("Error invoking get_entry_by_version: %v\n",err)
            os.Exit(1)
        }
    }

    fmt.Printf("Delete 'b'.\n")
    p = make([]interface{},0,10)
    p = append(p,"b")
    p = append(p,0)
    if _,err := dirc.Invoke_method("delete_entry",p); err == nil {
        fmt.Printf("Deleted 'b'\n")
    } else {
        fmt.Printf("Error invoking delete_entry: %v\n",err)
        os.Exit(1)
    }

    // fmt.Printf("Retrieve a list of all registered names, should have MTN contracts plus 'a,c,c' in it.\n")
    // p = make([]interface{},0,10)
    // p = append(p,0)
    // p = append(p,10)
    // if nl,err := dirc.Invoke_method("get_names",p); err == nil {
    //     fmt.Printf("Registered names %v\n",nl)
    // } else {
    //     fmt.Printf("Error invoking get_names: %v\n",err)
    //     os.Exit(1)
    // }

    fmt.Printf("Delete 'c' version 0.\n")
    p = make([]interface{},0,10)
    p = append(p,"c")
    p = append(p,0)
    if _,err := dirc.Invoke_method("delete_entry",p); err == nil {
        fmt.Printf("Deleted 'c' version 0.\n")
    } else {
        fmt.Printf("Error invoking delete_entry: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Delete 'c' version 1.\n")
    p = make([]interface{},0,10)
    p = append(p,"c")
    p = append(p,1)
    if _,err := dirc.Invoke_method("delete_entry",p); err == nil {
        fmt.Printf("Deleted 'c' version 1.\n")
    } else {
        fmt.Printf("Error invoking delete_entry: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Delete 'a'.\n")
    p = make([]interface{},0,10)
    p = append(p,"a")
    p = append(p,0)
    if _,err := dirc.Invoke_method("delete_entry",p); err == nil {
        fmt.Printf("Deleted 'a'\n")
    } else {
        fmt.Printf("Error invoking delete_entry: %v\n",err)
        os.Exit(1)
    }

    // fmt.Printf("Retrieve a list of all registered names, should be just the MTN platform entries.\n")
    // p = make([]interface{},0,10)
    // p = append(p,0)
    // p = append(p,10)
    // if nl,err := dirc.Invoke_method("get_names",p); err == nil {
    //     fmt.Printf("Registered names %v\n",nl)
    // } else {
    //     fmt.Printf("Error invoking get_names: %v\n",err)
    //     os.Exit(1)
    // }

    // ================= whisper directory tests ===================================
    // Find the whisper directory
    fmt.Printf("Retrieve contract for whisper directory.\n")
    p = make([]interface{},0,10)
    p = append(p,"whisper_directory")
    var wdaddr string
    if wda,err := dirc.Invoke_method("get_entry",p); err == nil {
        fmt.Printf("Contract Address is %v\n",wda)
        wdaddr = wda.(string)
    } else {
        fmt.Printf("Error invoking get_entry: %v\n",err)
        os.Exit(1)
    }

    // Establish the whisper directory contract
    wd := contract_api.SolidityContractFactory("whisper_directory")
    if _,err := wd.Load_contract(registry_owner, ""); err != nil {
        fmt.Printf("...terminating, could not load whisper directory contract: %v\n",err)
        os.Exit(1)
    }
    wd.Set_contract_address(wdaddr)

    fmt.Printf("Add an entry for a long string.\n")
    p = make([]interface{},0,10)
    p = append(p,"0x04bd30d6872ae1396d537195d76482d8828682673221d41c8420bd07d043b72851d4a7cb9a7d0bcb8360393e27396bc643a75ca7aaa65380becf68e4727257de57")
    if _,err := wd.Invoke_method("add_entry",p); err == nil {
        fmt.Printf("Added an entry.\n")
    } else {
        fmt.Printf("Error invoking whisper add_entry: %v\n",err)
        os.Exit(1)
    }

    start_timer = time.Now()
    for {
        fmt.Printf("Get your current entry, should be 0x04bd30d6872ae1396d537195d76482d8828682673221d41c8420bd07d043b72851d4a7cb9a7d0bcb8360393e27396bc643a75ca7aaa65380becf68e4727257de57.\n")
        p = make([]interface{},0,10)
        p = append(p,registry_owner)
        if wa,err := wd.Invoke_method("get_entry",p); err == nil {
            fmt.Printf("Received %v.\n",wa)
            if wa.(string) != "0x04bd30d6872ae1396d537195d76482d8828682673221d41c8420bd07d043b72851d4a7cb9a7d0bcb8360393e27396bc643a75ca7aaa65380becf68e4727257de57" {
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
            fmt.Printf("Error invoking whisper get_entry: %v\n",err)
            os.Exit(1)
        }
    }

    fmt.Printf("Get entry at address 0x00..001, it's not there.\n")
    p = make([]interface{},0,10)
    p = append(p,"0x0000000000000000000000000000000000000001")
    if wa,err := wd.Invoke_method("get_entry",p); err == nil {
        fmt.Printf("Received %v.\n",wa)
        if wa.(string) != "" {
            os.Exit(1)
        }
    } else {
        fmt.Printf("Error invoking whisper get_entry: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Add an entry for 0x0000deadbeef.\n")
    p = make([]interface{},0,10)
    p = append(p,"0x0000deadbeef")
    if _,err := wd.Invoke_method("add_entry",p); err == nil {
        fmt.Printf("Added an entry.\n")
    } else {
        fmt.Printf("Error invoking whisper add_entry: %v\n",err)
        os.Exit(1)
    }

    start_timer = time.Now()
    for {
        fmt.Printf("Get your current entry, should be 0x0000deadbeef.\n")
        p = make([]interface{},0,10)
        p = append(p,registry_owner)
        if wa,err := wd.Invoke_method("get_entry",p); err == nil {
            fmt.Printf("Received %v.\n",wa)
            if wa.(string) != "0x0000deadbeef" {
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
            fmt.Printf("Error invoking whisper get_entry: %v\n",err)
            os.Exit(1)
        }
    }

    fmt.Printf("Add an entry for a long string.\n")
    p = make([]interface{},0,10)
    p = append(p,"0x04bd30d6872ae1396d537195d76482d8828682673221d41c8420bd07d043b72851d4a7cb9a7d0bcb8360393e27396bc643a75ca7aaa65380becf68e4727257de57")
    if _,err := wd.Invoke_method("add_entry",p); err == nil {
        fmt.Printf("Added an entry.\n")
    } else {
        fmt.Printf("Error invoking whisper add_entry: %v\n",err)
        os.Exit(1)
    }

    start_timer = time.Now()
    for {
        fmt.Printf("Get your current entry, should be 0x04bd30d6872ae1396d537195d76482d8828682673221d41c8420bd07d043b72851d4a7cb9a7d0bcb8360393e27396bc643a75ca7aaa65380becf68e4727257de57.\n")
        p = make([]interface{},0,10)
        p = append(p,registry_owner)
        if wa,err := wd.Invoke_method("get_entry",p); err == nil {
            fmt.Printf("Received %v.\n",wa)
            if wa.(string) != "0x04bd30d6872ae1396d537195d76482d8828682673221d41c8420bd07d043b72851d4a7cb9a7d0bcb8360393e27396bc643a75ca7aaa65380becf68e4727257de57" {
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
            fmt.Printf("Error invoking whisper get_entry: %v\n",err)
            os.Exit(1)
        }
    }

    fmt.Printf("Update your entry with 0x000012345678.\n")
    p = make([]interface{},0,10)
    p = append(p,"0x000012345678")
    if _,err := wd.Invoke_method("add_entry",p); err == nil {
        fmt.Printf("Updated the entry.\n")
    } else {
        fmt.Printf("Error invoking whisper add_entry: %v\n",err)
        os.Exit(1)
    }

    start_timer = time.Now()
    for {
        fmt.Printf("Get your current entry, should be 0x000012345678.\n")
        p = make([]interface{},0,10)
        p = append(p,registry_owner)
        if wa,err := wd.Invoke_method("get_entry",p); err == nil {
            fmt.Printf("Received %v.\n",wa)
            if wa.(string) != "0x000012345678" {
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
            fmt.Printf("Error invoking whisper get_entry: %v\n",err)
            os.Exit(1)
        }
    }

    fmt.Printf("Delete your entry.\n")
    if _,err := wd.Invoke_method("delete_entry",nil); err == nil {
        fmt.Printf("Deleted the entry.\n")
    } else {
        fmt.Printf("Error invoking whisper add_entry: %v\n",err)
        os.Exit(1)
    }

    start_timer = time.Now()
    for {
        fmt.Printf("Get your current entry, should be empty string.\n")
        p = make([]interface{},0,10)
        p = append(p,registry_owner)
        if wa,err := wd.Invoke_method("get_entry",p); err == nil {
            fmt.Printf("Received %v.\n",wa)
            if wa.(string) != "" {
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
            fmt.Printf("Error invoking whisper get_entry: %v\n",err)
            os.Exit(1)
        }
    }

    // Find all events related to the directory test in the blockchain and dump them into the output.

    log.Printf("Dumping blockchain event data for contract %v.\n",dirc.Get_contract_address())
    result, out, err := "", "", error(nil)
    var rpcResp *rpcResponse = new(rpcResponse)

    params := make(map[string]string)
    params["address"] = dirc.Get_contract_address()
    params["fromBlock"] = "0x1"

    if out, err = dirc.Call_rpc_api("eth_newFilter", params); err == nil {
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
    if out, err = dirc.Call_rpc_api("eth_getFilterLogs", result); err == nil {
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
            format_dirc_event(ix, ev);
        }
    }

    // Find all events related to the whisper directory test in the blockchain and dump them into the output.

    log.Printf("Dumping blockchain event data for contract %v.\n",wd.Get_contract_address())
    result, out, err = "", "", error(nil)
    rpcResp = new(rpcResponse)

    params = make(map[string]string)
    params["address"] = wd.Get_contract_address()
    params["fromBlock"] = "0x1"

    if out, err = wd.Call_rpc_api("eth_newFilter", params); err == nil {
        if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
            if rpcResp.Error.Message != "" {
                log.Printf("eth_newFilter returned an error: %v.\n", rpcResp.Error.Message)
            } else {
                result = rpcResp.Result.(string)
                // log.Printf("Event id: %v.\n",result)
            }
        }
    }

    rpcFilterResp = new(rpcGetFilterChangesResponse)
    if out, err = wd.Call_rpc_api("eth_getFilterLogs", result); err == nil {
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
            format_wd_event(ix, ev);
        }
    }
    
    // Find all events related to the whisper directory in the blockchain and dump them into the output.

    log.Printf("Dumping blockchain event data for %v using contract %v.\n",registry_owner,wd.Get_contract_address())
    result, out, err = "", "", error(nil)
    rpcResp = new(rpcResponse)

    fparams := make(map[string]interface{})
    fparams["address"] = wd.Get_contract_address()
    fparams["fromBlock"] = "0x1"
    topics := make([]interface{},0,10)
    topics = append(topics, nil)
    topics = append(topics,"0x"+strings.Repeat("0", (64-len(registry_owner))) + registry_owner)
    fparams["topics"] = topics

    if out, err = wd.Call_rpc_api("eth_newFilter", fparams); err == nil {
        if err = json.Unmarshal([]byte(out), rpcResp); err == nil {
            if rpcResp.Error.Message != "" {
                log.Printf("eth_newFilter returned an error: %v.\n", rpcResp.Error.Message)
            } else {
                result = rpcResp.Result.(string)
                // log.Printf("Event id: %v.\n",result)
            }
        }
    }

    rpcFilterResp = new(rpcGetFilterChangesResponse)
    if out, err = wd.Call_rpc_api("eth_getFilterLogs", result); err == nil {
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
            format_wd_event(ix, ev);
        }
    }

    fmt.Println("Terminating directory test client")
}

func format_dirc_event(ix int, ev rpcFilterChanges) {
    // These event string correspond to event codes from the container_executor contract
    dirc_add_ev := "0x0000000000000000000000000000000000000000000000000000000000000000"
    dirc_del_ev := "0x0000000000000000000000000000000000000000000000000000000000000001"

    if ev.Topics[0] == dirc_add_ev {
        log.Printf("|%03d| Entry added by %v version %v for %v\n",ix,ev.Topics[1],ev.Topics[2],ev.Topics[3]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
    } else if ev.Topics[0] == dirc_del_ev {
        log.Printf("|%03d| Entry deleted by %v version %v for %v\n",ix,ev.Topics[1],ev.Topics[2],ev.Topics[3]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
    } else {
        log.Printf("|%03d| Unknown event code in first topic slot.\n")
        log.Printf("Raw log entry:\n%v\n\n",ev)
    }
}

func format_wd_event(ix int, ev rpcFilterChanges) {
    // These event string correspond to event codes from the container_executor contract
    wd_add_ev := "0x0000000000000000000000000000000000000000000000000000000000000000"
    wd_del_ev := "0x0000000000000000000000000000000000000000000000000000000000000001"

    if ev.Topics[0] == wd_add_ev {
        log.Printf("|%03d| Entry added by %v\n",ix,ev.Topics[1]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
    } else if ev.Topics[0] == wd_del_ev {
        log.Printf("|%03d| Entry deleted by %v\n",ix,ev.Topics[1]);
        log.Printf("Data: %v\n",ev.Data);
        log.Printf("Block: %v\n\n",ev.BlockNumber);
    } else {
        log.Printf("|%03d| Unknown event code in first topic slot.\n")
        log.Printf("Raw log entry:\n%v\n\n",ev)
    }
}

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
