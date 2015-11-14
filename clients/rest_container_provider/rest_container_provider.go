package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    //"io/ioutil"
    "math/rand"
    "net/http"
    "os"
    //"repo.hovitos.engineering/MTN/go-solidity/contract_api"
    "strconv"
    "time"
    )

func main() {
    fmt.Println("Starting REST container provider client")

    if len(os.Args) < 4 {
        fmt.Printf("...terminating, only %v parameters were passed.\n",len(os.Args))
        os.Exit(1)
    }

    err := error(nil)

    whisper_account := os.Args[1]
    fmt.Printf("using whisper account %v\n",whisper_account)
    container_owner := os.Args[2]
    fmt.Printf("using account %v\n",container_owner)
    tx_lost_delay_toleration,_ := strconv.Atoi(os.Args[2])

    // ---------------------- Start of one time init code ------------------------
    //

    // Check glensung bacon balance
    
    pr := BankResponseWith{}
    err = invoke_rest("GET", "bank?loans=true", nil, pr)
    if err != nil {
        fmt.Printf("Error getting bank balance:%v\n",err)
        os.Exit(1)
    }
    bal := pr.Tokens
    fmt.Printf("Owner bacon balance is:%v\n",bal)

    if bal == 0 {
        // Get a loan
        pr := BankPutRequest{}
        pr.Amount = 1000
        body,err := json.Marshal(pr)
        if err != nil {
            fmt.Printf("Error marshalling loan request:%v\n",err)
            os.Exit(1)
        }
        err = invoke_rest("PUT", "bank", body, nil)
        if err != nil {
            fmt.Printf("Error obtaining a loan:%v\n",err)
            os.Exit(1)
        }

        // Loop until the loan is granted. The PUT REST API will timeout
        // after a minute if the transaction doesn't run.
        start_timer := time.Now()
        for bal == 0 {
            delta := time.Now().Sub(start_timer).Seconds()
            if int(delta) < tx_lost_delay_toleration {
                time.Sleep(5000*time.Millisecond)
                pr := BankResponseWith{}
                err = invoke_rest("GET", "bank?loans=true", nil, pr)
                if err != nil {
                    fmt.Printf("Error getting bank balance:%v\n",err)
                    os.Exit(1)
                }
                bal = pr.Tokens
            } else {
                fmt.Printf("Loan never came through.\n")
                os.Exit(1)
            }
        }
        fmt.Printf("Owner bacon balance is:%v\n",bal)
    }

    //
    // ------------------- End of one time initialization ------------------------

    // ------------------- Start of worker loop ----------------------------------
    //

    // Run the work loop until there are no more devices to contract with
    first_device := false
    last_device := false
    for !last_device {

        pr := RegistryResponse{}
        err = invoke_rest("GET", "registry", nil, pr)
        if err != nil {
            fmt.Printf("Error getting device list from registry:%v\n",err)
            os.Exit(1)
        }

        if len(pr) > 0 {
            if !first_device {
                first_device = true
            }
        } else {
            if first_device {
                last_device = true
            }
        }

        for _,device := range pr {

            // Try to setup an agreement with this device
            dr := DeviceResponse{}
            err = invoke_rest("GET", "device?address="+device.Address, nil, dr)
            if err != nil {
                fmt.Printf("Error getting device info:%v\n",err)
                os.Exit(1)
            }

            fmt.Printf("Checking to see if device is already running a container.\n")
            if dr.Agreement.AgreementId == "" {
                fmt.Printf("Device is available. Tell it to run a container.\n")

                // Try to enter agreement with the device
                dp := DevicePutRequest{}
                dp.Amount = rand.Intn(4)+1
                dp.Address = device.Address
                dp.AgreementId = "abcdefghijklmnopqrstuvwxyz"
                dp.Whisper = whisper_account
                body,err := json.Marshal(dp)
                if err != nil {
                    fmt.Printf("Error marshalling request to enter an agreement:%v\n",err)
                    os.Exit(1)
                }
                err = invoke_rest("PUT", "device", body, nil)
                if err != nil {
                    fmt.Printf("Error entering a new agreement:%v\n",err)
                    os.Exit(1)
                }

                // Make sure agreement is entered, if not try again later
                agreement_set := false
                start_timer := time.Now()
                for !agreement_set {
                    delta := time.Now().Sub(start_timer).Seconds()
                    if int(delta) < tx_lost_delay_toleration {
                        time.Sleep(5000*time.Millisecond)
                        dr1 := DeviceResponse{}
                        err = invoke_rest("GET", "device?address="+device.Address, nil, dr1)
                        if err != nil {
                            fmt.Printf("Error getting device info:%v\n",err)
                            os.Exit(1)
                        }
                        if dr1.Agreement.AgreementId == "abcdefghijklmnopqrstuvwxyz" {
                            fmt.Printf("Device has our agreement Id.\n")
                            agreement_set = true
                        }
                    } else {
                        fmt.Printf("Agreement was never picked up.\n")
                        break
                    }
                }

                // Our agreement ID reached the device contract, now we wait for the device to
                // decide what to do; accept or cancel.
                if agreement_set {
                    fmt.Printf("Waiting for device to accept proposal, or cancel.\n")
                    agreement_reached := false
                    cancelled := false
                    dr1 := DeviceResponse{}
                    start_timer := time.Now()
                    for !agreement_reached && !cancelled {
                        delta := time.Now().Sub(start_timer).Seconds()
                        if int(delta) < (tx_lost_delay_toleration*5) {
                            time.Sleep(5000*time.Millisecond)
                            err = invoke_rest("GET", "device?address="+device.Address, nil, dr1)
                            if err != nil {
                                fmt.Printf("Error getting device info:%v\n",err)
                                os.Exit(1)
                            }
                            if dr1.Agreement.Counterparty_accepted {
                                fmt.Printf("Device has agreed.\n")
                                agreement_reached = true
                            }
                            if dr1.Agreement.AgreementId == "" {
                                fmt.Printf("Device has cancelled.\n")
                                cancelled = true
                            }
                        } else {
                            fmt.Printf("Timeout waiting for device to decide on agreement.\n")
                            break
                        }
                    }
                    if cancelled {
                        fmt.Printf("Device cancelled proposal.\n")
                        continue
                    } else if !agreement_reached && dr1.Agreement.Counterparty != "" {
                        // We will cancel
                        fmt.Printf("Device has neither accepted nor cancelled, we will cancel.\n")
                        cr := DevicePostRequest{}
                        cr.Action = "cancel"
                        cr.Device = device.Address
                        cr.Proposer = container_owner
                        cr.Counterparty = dr1.Agreement.Counterparty
                        body,err := json.Marshal(cr)
                        if err != nil {
                            fmt.Printf("Error marshalling cancel request:%v\n",err)
                            os.Exit(1)
                        }
                        err = invoke_rest("POST", "device", body, nil)
                        if err != nil {
                            fmt.Printf("Error cancelling agreement:%v\n",err)
                            os.Exit(1)
                        }
                        continue
                    } else if agreement_reached {
                        // counterparty has accepted, we will now accept
                        fmt.Printf("Device has accepted, now its our turn.\n")







                    } else {
                        fmt.Printf("Device must have cancelled.\n")
                        continue
                    }

                } // agreement Id was set onto device
            } // device is available
        } // for each device

    //             p := make([]interface{},0,10)
    //             p = append(p,"whisper1")
    //             p = append(p,"agreement1")
    //             p = append(p,(rand.Intn(4))+1)
    //             if _,err = sc.Invoke_method("new_container",p); err != nil {
    //                 fmt.Printf("...terminating, could not initiate new container agreement: %v\n",err)
    //                 os.Exit(1)
    //             }

    //             fmt.Printf("Waiting for acceptance of proposal.\n")
    //             var received_codes []uint64
    //             received_codes,_ = bank.Wait_for_event([]uint64{counterparty_accepted_event_code,escrow_cancelled_event_code}, sc.Get_contract_address())
    //             fmt.Printf("Received event codes: %v\n",received_codes)

    //             found_cancel := false
    //             for _,ev := range received_codes {
    //                 if ev == escrow_cancelled_event_code {
    //                     found_cancel = true
    //                     fmt.Printf("Received cancel escrow.\n")
    //                 }
    //             }

    //             if found_cancel {
    //                 fmt.Printf("Try again.\n")
    //             } else {
    //                 fmt.Printf("Voting for proposal.\n")
    //                 var device_owner interface{}
    //                 if device_owner,_ = sc.Invoke_method("get_owner",nil); err != nil {
    //                     fmt.Printf("...terminating, could not get_owner on self: %v\n",err)
    //                     os.Exit(1)
    //                 }
    //                 p = make([]interface{},0,10)
    //                 p = append(p,device_owner.(string))
    //                 p = append(p,sc.Get_contract_address())
    //                 p = append(p,true)
    //                 if _,_ = bank.Invoke_method("proposer_vote",p); err != nil {
    //                     fmt.Printf("...terminating, could not vote for proposal: %v\n",err)
    //                     os.Exit(1)
    //                 }

    //                 fmt.Printf("Waiting for completion of container execution.\n")
    //                 received_codes,_ = sc.Wait_for_event([]uint64{execution_complete_event_code,container_rejected_event_code},sc.Get_contract_address())
    //                 fmt.Printf("Received event codes: %v\n",received_codes)

    //                 fmt.Printf("The device is available again.\n")
    //             }
    //         } else {
    //             fmt.Printf("Error, executor is busy.\n")
                
    //         }
    //     }
    //     // Check glensung bacon balance
    //     var bal interface{}
    //     if bal,err = bank.Invoke_method("account_balance",nil); err != nil {
    //         fmt.Printf("...terminating, could not get token balance: %v\n",err)
    //         os.Exit(1)
    //     }
    //     fmt.Printf("Owner bacon balance is:%v\n",bal)

        // short delay
        time.Sleep(5000*time.Millisecond)

    } // there are devices still in the registry

    //
    // ------------------- End of worker loop ------------------------------------

    fmt.Println("Terminating REST container provider.")
}

// ----------------- REST structs ---------------------------------------------

type BankResponseWith struct {
    Tokens uint64 `json:"tokens"`
    Loans []loan `json:"loans"`
}

type loan struct {
    Id string `json:"id"`
    Balance uint64 `json:"balance"`
}

type BankPutRequest struct {
    Amount int `json:"amount"`
}

type RegistryResponse []device_address

type device_address struct {
    Address string `json:"address"`
}

type DeviceResponse struct {
    Agreement agreement `json:"agreement"`
    Attributes []map[string]interface{} `json:"attributes"`
}

type agreement struct {
    Proposer string `json:"proposer"`
    Counterparty string `json:"counterparty"`
    AgreementId string `json:"agreementId"`
    Counterparty_accepted bool `json:"counterparty-accepted"`
    Proposer_accepted bool `json:"proposer-accepted"`
    Escrow_amount uint64 `json:"escrow-amount"`
}

type DevicePutRequest struct {
    Address string `json:"address"`
    AgreementId string `json:"agreementId"`
    Whisper string `json:"whisper"`
    Amount int `json:"amount"`
}

type DevicePostRequest struct {
    Action string `json:"action"`
    Device string `json:"device"`
    Proposer string `json:"proposer"`
    Counterparty string `json:"counterparty"`
}


func invoke_rest(method string, url string, body []byte, outstruct interface{}) error {
    var base_url = "http://localhost:8000/mtn/marketplace/v1/"
    req, err := http.NewRequest(method, base_url + url, bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    rawresp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer rawresp.Body.Close()
    if outstruct != nil {
        err = json.NewDecoder(rawresp.Body).Decode(&outstruct)
    }

    fmt.Println("response Status:", rawresp.Status)
    // fmt.Println("response Headers:", rawresp.Header)
    return err
}
