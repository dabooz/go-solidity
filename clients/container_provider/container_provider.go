package main

import (
    "fmt"
    "github.com/open-horizon/go-solidity/contract_api"
    "math/rand"
    "os"
    "time"
    )

func main() {
    fmt.Println("Starting container provider client")

    if len(os.Args) < 3 {
        fmt.Printf("...terminating, only %v parameters were passed.\n",len(os.Args))
        os.Exit(1)
    }

    err := error(nil)

    dir_contract := os.Args[1]
    container_owner := os.Args[2]

    //var new_container_event_code uint64 = 1
    var execution_complete_event_code uint64 = 2
    var container_rejected_event_code uint64 = 3

    var escrow_cancelled_event_code uint64 = 6
    var counterparty_accepted_event_code uint64 = 7

    // ---------------------- Start of one time init code ------------------------
    //

    // Establish the directory contract
    dirc := contract_api.SolidityContractFactory("directory")
    if _,err := dirc.Load_contract(container_owner, ""); err != nil {
        fmt.Printf("...terminating, could not load directory contract: %v\n",err)
        os.Exit(1)
    }
    dirc.Set_contract_address(dir_contract)

    // Find the token_bank contract
    var tbaddr interface{}
    p := make([]interface{},0,10)
    p = append(p,"token_bank")
    if tbaddr,err = dirc.Invoke_method("get_entry",p); err != nil {
        fmt.Printf("...terminating, could not find token_bank in directory: %v\n",err)
        os.Exit(1)
    }
    fmt.Printf("token_bank addr is %v\n",tbaddr)

    // Establish the token_bank contract
    bank := contract_api.SolidityContractFactory("token_bank")
    if _,err := bank.Load_contract(container_owner, ""); err != nil {
        fmt.Printf("...terminating, could not load token_bank contract: %v\n",err)
        os.Exit(1)
    }
    bank.Set_contract_address(tbaddr.(string))

    // Check glensung bacon balance
    var bal interface{}
    if bal,err = bank.Invoke_method("account_balance",nil); err != nil {
        fmt.Printf("...terminating, could not get token balance: %v\n",err)
        os.Exit(1)
    }
    fmt.Printf("Owner bacon balance is:%v\n",bal)

    if bal.(uint64) == 0 {
        p := make([]interface{},0,10)
        p = append(p,1000)
        if _,err = bank.Invoke_method("obtain_loan",p); err != nil {
            fmt.Printf("...terminating, could not get a loan: %v\n",err)
            os.Exit(1)
        }
        var bal interface{}
        if bal,err = bank.Invoke_method("account_balance",nil); err != nil {
            fmt.Printf("...terminating, could not get token balance: %v\n",err)
            os.Exit(1)
        }
        fmt.Printf("Owner bacon balance is now:%v\n",bal)
    }

    // Find the device registry contract
    var draddr interface{}
    p = make([]interface{},0,10)
    p = append(p,"device_registry")
    if draddr,err = dirc.Invoke_method("get_entry",p); err != nil {
        fmt.Printf("...terminating, could not find device_registry in directory: %v\n",err)
        os.Exit(1)
    }
    fmt.Printf("device_registry addr is %v\n",draddr)

    // Establish the device_registry contract
    dr := contract_api.SolidityContractFactory("device_registry")
    if _,err := dr.Load_contract(container_owner, ""); err != nil {
        fmt.Printf("...terminating, could not load device_registry contract: %v\n",err)
        os.Exit(1)
    }
    dr.Set_contract_address(draddr.(string))

    //
    // ------------------- End of one time initialization ------------------------

    // ------------------- Start of worker loop ----------------------------------
    //
 
    // Establish the device contract
    sc := contract_api.SolidityContractFactory("container_executor")
    if _,err := sc.Load_contract(container_owner, ""); err != nil {
        fmt.Printf("...terminating, could not load container_executor contract: %v\n",err)
        os.Exit(1)
    }

    // Run the work loop until there are no more devices to contract with
    first_device := false
    last_device := false
    for !last_device {

        // Find a device in the registry, based on the attributes you care about.
        var devices interface{}
        p := make([]interface{},0,10)
        p = append(p,0)     // start index
        p = append(p,10)     // end index
        p2 := make([]string,0,10)
        p2 = append(p2,"arch")
        p2 = append(p2,"armhf")
        p = append(p,p2)
        if devices,err = dr.Invoke_method("find_by_attributes",p); err != nil {
            fmt.Printf("...terminating, could not find device by attributes: %v\n",err)
            os.Exit(1)
        }
        fmt.Printf("Returned device addresses %v.\n",devices)
        device_array := devices.([]string)
        //device_addr := device_array[0]

        if len(device_array) > 0 {
            if !first_device {
                first_device = true
            }
        } else {
            if first_device {
                last_device = true
            }
        }

        for _,device_addr := range device_array {

            // Set contract address for specfic device
            sc.Set_contract_address(device_addr)

            // Try to setup an agreement with this device
            fmt.Printf("Checking to see if device is already running a container.\n")
            var in_contract interface{}
            if in_contract,err = sc.Invoke_method("in_contract",nil); err!= nil {
                fmt.Printf("...terminating, could not get contract status: %v\n",err)
                os.Exit(1)
            }
            var has_agreement bool
            has_agreement = in_contract.(bool)
            if has_agreement == false {
                fmt.Printf("Device is available. Tell it to run a container.\n")

                p := make([]interface{},0,10)
                p = append(p,"whisper1")
                p = append(p,"agreement1")
                p = append(p,(rand.Intn(4))+1)
                if _,err = sc.Invoke_method("new_container",p); err != nil {
                    fmt.Printf("...terminating, could not initiate new container agreement: %v\n",err)
                    os.Exit(1)
                }

                fmt.Printf("Waiting for acceptance of proposal.\n")
                var received_codes []uint64
                received_codes,_ = bank.Wait_for_event([]uint64{counterparty_accepted_event_code,escrow_cancelled_event_code}, sc.Get_contract_address())
                fmt.Printf("Received event codes: %v\n",received_codes)

                found_cancel := false
                for _,ev := range received_codes {
                    if ev == escrow_cancelled_event_code {
                        found_cancel = true
                        fmt.Printf("Received cancel escrow.\n")
                    }
                }

                if found_cancel {
                    fmt.Printf("Try again.\n")
                } else {
                    fmt.Printf("Voting for proposal.\n")
                    var device_owner interface{}
                    if device_owner,_ = sc.Invoke_method("get_owner",nil); err != nil {
                        fmt.Printf("...terminating, could not get_owner on self: %v\n",err)
                        os.Exit(1)
                    }
                    p = make([]interface{},0,10)
                    p = append(p,device_owner.(string))
                    p = append(p,sc.Get_contract_address())
                    p = append(p,true)
                    if _,_ = bank.Invoke_method("proposer_vote",p); err != nil {
                        fmt.Printf("...terminating, could not vote for proposal: %v\n",err)
                        os.Exit(1)
                    }

                    fmt.Printf("Waiting for completion of container execution.\n")
                    received_codes,_ = sc.Wait_for_event([]uint64{execution_complete_event_code,container_rejected_event_code},sc.Get_contract_address())
                    fmt.Printf("Received event codes: %v\n",received_codes)

                    fmt.Printf("The device is available again.\n")
                }
            } else {
                fmt.Printf("Error, executor is busy.\n")
                
            }
        }
        // Check glensung bacon balance
        var bal interface{}
        if bal,err = bank.Invoke_method("account_balance",nil); err != nil {
            fmt.Printf("...terminating, could not get token balance: %v\n",err)
            os.Exit(1)
        }
        fmt.Printf("Owner bacon balance is:%v\n",bal)

        // short delay
        if len(device_array) == 0 {
            time.Sleep(10000*time.Millisecond)
        }

    }

    //
    // ------------------- End of worker loop ------------------------------------

    fmt.Println("Terminating client")
}

