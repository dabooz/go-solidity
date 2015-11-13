package main

import (
    "fmt"
    "repo.hovitos.engineering/MTN/go-solidity/contract_api"
    )

func main() {
    fmt.Println("Starting client1")

    sc := contract_api.SolidityContractFactory("container_executor")
    fmt.Printf("Solidity Contract object: %v\n",sc)

    out,err := sc.Deploy_contract("0x3cf988a4da55af5c03ba8f32210436041a4b159d", "http://158.85.109.248:8545")

    if err == nil {
        fmt.Printf("Contract %v\n",out)
        if owner,err := sc.Invoke_method("get_owner",nil); err == nil {
            fmt.Printf("Owner %v\n",owner)
        } else {
            fmt.Printf("error %v\n",err)
        }

        if cid,err := sc.Invoke_method("get_container_id",nil); err == nil {
            fmt.Printf("CID %v\n",cid)
        } else {
            fmt.Printf("error %v\n",err)
        }

        p := make([]interface{},0,10)
        p = append(p,111)
        p = append(p,"seq_http:latest")
        p = append(p,"abcdefg")
        if nc,err := sc.Invoke_method("new_container",p); err == nil {
            fmt.Printf("Called new container %v\n",nc)
        } else {
            fmt.Printf("error %v\n",err)
        }

        if cname,err := sc.Invoke_method("get_container_name",nil); err == nil {
            fmt.Printf("CName %v\n",cname)
        } else {
            fmt.Printf("error %v\n",err)
        }

    } else {
        fmt.Printf("error %v\n",err)
    }

    fmt.Println("Terminating client1")

}
