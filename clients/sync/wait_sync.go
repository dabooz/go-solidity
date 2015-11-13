package main 

import (
    "fmt"
    "encoding/json"
    "repo.hovitos.engineering/MTN/go-solidity/contract_api"
    "os"
    "strconv"
    "time"
)

type rpcResponse struct {
    Id string `json:"id"`
    Version string `json:"jsonrpc"`
    Result interface{} `json:"result"`
    Error struct {
        Code int `json:"code"`
        Message string `json:"message"`
    } `json:"error"`
}

type RPCError struct {
    msg string
}
func (e *RPCError) Error() string { if e != nil {return e.msg} else {return ""} }



func main() {
    fmt.Println("Waiting for node to complete the sync process.")

    err := error(nil)
    var res string
    var rpcResp *rpcResponse = new(rpcResponse)
    net_done := false
    block_done := false
    sync_done := false

    poll_wait := 5
    if len(os.Args) < 2 {
        fmt.Printf("Polling wait time not specified, using %v\n",poll_wait)
    }
    poll_wait,_ = strconv.Atoi(os.Args[1])

    for !net_done {
        sc := contract_api.SolidityContractFactory("dummy")
        if res,err = sc.Call_rpc_api("net_peerCount",nil); err != nil {
            fmt.Printf("Treating, %v ,as a temporary error, retrying.\n",err)
            time.Sleep(time.Duration(poll_wait)*1000*time.Millisecond)
            continue
        }
        if err = json.Unmarshal([]byte(res),rpcResp); err != nil {
            fmt.Printf("Treating, %v ,as a permanent error.\n",err)
            os.Exit(1)
        }

        if rpcResp.Error.Message != "" {
            err = &RPCError{fmt.Sprintf("RPC invocation returned an error: %v.",rpcResp.Error.Message)}
            fmt.Printf("Treating, %v ,as a permanent error.\n",err)
            os.Exit(1)
        } else {
            fmt.Printf("netPeering result: %v\n",rpcResp)
            switch rpcResp.Result.(type) {
                case string:
                    if rpcResp.Result != "0x0" {
                        net_done = true
                    } else {
                        fmt.Printf("Still syncing...\n")
                        time.Sleep(time.Duration(poll_wait)*1000*time.Millisecond)
                        continue
                    }
                default:
                    fmt.Printf("Still syncing...\n")
                    time.Sleep(time.Duration(poll_wait)*1000*time.Millisecond)
                    continue
            }
        }
    }

    for !block_done {
        sc := contract_api.SolidityContractFactory("dummy")
        if res,err = sc.Call_rpc_api("eth_blockNumber",nil); err != nil {
            fmt.Printf("Treating, %v ,as a temporary error, retrying.\n",err)
            time.Sleep(time.Duration(poll_wait)*1000*time.Millisecond)
            continue
        }
        if err = json.Unmarshal([]byte(res),rpcResp); err != nil {
            fmt.Printf("Treating, %v ,as a permanent error.\n",err)
            os.Exit(1)
        }

        if rpcResp.Error.Message != "" {
            err = &RPCError{fmt.Sprintf("RPC invocation returned an error: %v.",rpcResp.Error.Message)}
            fmt.Printf("Treating, %v ,as a permanent error.\n",err)
            os.Exit(1)
        } else {
            fmt.Printf("blockNumber result: %v\n",rpcResp)
            switch rpcResp.Result.(type) {
                case string:
                    if rpcResp.Result != "0x0" {
                        block_done = true
                    } else {
                        fmt.Printf("Still syncing...\n")
                        time.Sleep(time.Duration(poll_wait)*1000*time.Millisecond)
                        continue
                    }
                default:
                    fmt.Printf("Still syncing...\n")
                    time.Sleep(time.Duration(poll_wait)*1000*time.Millisecond)
                    continue
            }
        }
    }


    for !sync_done {
        sc := contract_api.SolidityContractFactory("dummy")
        if res,err = sc.Call_rpc_api("eth_syncing",nil); err == nil {
            if err = json.Unmarshal([]byte(res),rpcResp); err == nil {
                if rpcResp.Error.Message != "" {
                    err = &RPCError{fmt.Sprintf("RPC invocation returned an error: %v.",rpcResp.Error.Message)}
                    fmt.Printf("Treating, %v ,as a permanent error.\n",err)
                    os.Exit(1)
                } else {
                    fmt.Printf("syncing result: %v\n",rpcResp)
                    switch rpcResp.Result.(type) {
                        case bool:
                            if rpcResp.Result == false {
                                sync_done = true
                            }
                        default:
                            fmt.Printf("Still syncing...\n")
                            time.Sleep(time.Duration(poll_wait)*1000*time.Millisecond)
                    }
                }
            } else {
                fmt.Printf("Treating, %v ,as a permanent error.\n",err)
                os.Exit(1)
            }
        } else {
            fmt.Printf("Treating, %v ,as a temporary error, retrying.\n",err)
            time.Sleep(time.Duration(poll_wait)*1000*time.Millisecond)
        }
    }

    fmt.Println("Node is synchronized.")
    os.Exit(0)

}