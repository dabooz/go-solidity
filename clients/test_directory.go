package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "repo.hovitos.engineering/MTN/go-solidity/contract_api"
    "net/http"
    "os"
    )

func main() {
    fmt.Println("Starting directory client")

    registry_owner := "0x759f4023b438d995dc37448314179c2a92ffda10"
    contract_name := "boozutreg"

    var buf []byte

    //var dat map[string]interface{}

    // Remove the registry contract from the web directory
    bs_site := "https://torrent.mtn.hovitos.engineering/marketplace/contract/"+contract_name
    req, _ := http.NewRequest("DELETE", bs_site, bytes.NewBuffer(buf))
    req.Header.Add("content-type", "application/json")
    client := &http.Client{}
    resp,_ := client.Do(req)
    defer resp.Body.Close()
    fmt.Printf("Old registration deleted.\n")

    // Deploy the registry contract
    fmt.Printf("Deploying Directory instance.\n")
    sc := contract_api.SolidityContractFactory("directory")
    _,err := sc.Deploy_contract(registry_owner, "http://158.85.109.248:8545")

    if err == nil {
        fmt.Printf("Directory deployed.\n")

        // Test to make sure the registry contract is invokable
        fmt.Printf("Retrieve contract for name 'a', should be zeroes.\n")
        p := make([]interface{},0,10)
        p = append(p,"a")
        if caddr,err := sc.Invoke_method("get_entry",p); err == nil {
            fmt.Printf("Contract Address is %v\n",caddr)
        } else {
            fmt.Printf("Error invoking get_entry: %v\n",err)
            os.Exit(1)
        }

        fmt.Printf("Retrieve a list of all registered names, should be empty.\n")
        p = make([]interface{},0,10)
        p = append(p,0)
        p = append(p,1)
        if nl,err := sc.Invoke_method("get_names",p); err == nil {
            fmt.Printf("Registered names %v\n",nl)
        } else {
            fmt.Printf("Error invoking get_names: %v\n",err)
            os.Exit(1)
        }

        // Save the device contract address in the web directory so that Glensung can find it
        fmt.Printf("Registering contract in web registry.\n")
        bs_site = "https://torrent.mtn.hovitos.engineering/marketplace/contract"
        body := make(map[string]interface{})
        body[contract_name] = sc.Get_contract_address()
        jsonBytes, _ := json.Marshal(body)
        req,_ = http.NewRequest("POST", bs_site, bytes.NewBuffer(jsonBytes))
        req.Header.Add("content-type", "application/json")
        client = &http.Client{}
        resp,_ = client.Do(req)
        defer resp.Body.Close()
        fmt.Printf("Directory contract added to web directory.\n")

        fmt.Printf("Register 'a' with address 0x0000000000000000000000000000000000000010.\n")
        p = make([]interface{},0,10)
        p = append(p,"a")
        p = append(p,"0x0000000000000000000000000000000000000010")
        if _,err := sc.Invoke_method("add_entry",p); err == nil {
            fmt.Printf("Registered 'a'.\n")
        } else {
            fmt.Printf("Error invoking add_entry: %v\n",err)
            os.Exit(1)
        }

        fmt.Printf("Retrieve 'a', should have address 10.\n")
        p = make([]interface{},0,10)
        p = append(p,"a")
        if aa,err := sc.Invoke_method("get_entry",p); err == nil {
            fmt.Printf("Retrieved 'a', is %v.\n",aa)
        } else {
            fmt.Printf("Error invoking add_entry: %v\n",err)
            os.Exit(1)
        }

        fmt.Printf("Retrieve owner of 'a', should be %v.\n",registry_owner)
        p = make([]interface{},0,10)
        p = append(p,"a")
        if aa,err := sc.Invoke_method("get_entry_owner",p); err == nil {
            fmt.Printf("Retrieved owner of 'a' %v.\n",aa)
        } else {
            fmt.Printf("Error invoking add_entry: %v\n",err)
            os.Exit(1)
        }

        fmt.Printf("Retrieve a list of all registered names, should have 'a' in it.\n")
        p = make([]interface{},0,10)
        p = append(p,0)
        p = append(p,1)
        if nl,err := sc.Invoke_method("get_names",p); err == nil {
            fmt.Printf("Registered names %v\n",nl)
        } else {
            fmt.Printf("Error invoking get_names: %v\n",err)
            os.Exit(1)
        }

        fmt.Printf("Register 'b' with address 0x0000000000000000000000000000000000000011.\n")
        p = make([]interface{},0,11)
        p = append(p,"b")
        p = append(p,"0x0000000000000000000000000000000000000011")
        if _,err := sc.Invoke_method("add_entry",p); err == nil {
            fmt.Printf("Registered 'b'.\n")
        } else {
            fmt.Printf("Error invoking add_entry: %v\n",err)
            os.Exit(1)
        }

        fmt.Printf("Register 'c' with address 0x0000000000000000000000000000000000000012.\n")
        p = make([]interface{},0,11)
        p = append(p,"c")
        p = append(p,"0x0000000000000000000000000000000000000012")
        if _,err := sc.Invoke_method("add_entry",p); err == nil {
            fmt.Printf("Registered 'c'.\n")
        } else {
            fmt.Printf("Error invoking add_entry: %v\n",err)
            os.Exit(1)
        }

        fmt.Printf("Retrieve a list of all registered names, should have 'a,b,c' in it.\n")
        p = make([]interface{},0,10)
        p = append(p,0)
        p = append(p,2)
        if nl,err := sc.Invoke_method("get_names",p); err == nil {
            fmt.Printf("Registered names %v\n",nl)
        } else {
            fmt.Printf("Error invoking get_names: %v\n",err)
            os.Exit(1)
        }

        fmt.Printf("Delete 'b'.\n")
        p = make([]interface{},0,10)
        p = append(p,"b")
        if _,err := sc.Invoke_method("delete_entry",p); err == nil {
            fmt.Printf("Deleted 'b'\n")
        } else {
            fmt.Printf("Error invoking delete_entry: %v\n",err)
            os.Exit(1)
        }

        fmt.Printf("Retrieve a list of all registered names, should have 'a,c' in it.\n")
        p = make([]interface{},0,10)
        p = append(p,0)
        p = append(p,2)
        if nl,err := sc.Invoke_method("get_names",p); err == nil {
            fmt.Printf("Registered names %v\n",nl)
        } else {
            fmt.Printf("Error invoking get_names: %v\n",err)
            os.Exit(1)
        }

        fmt.Printf("Delete 'c'.\n")
        p = make([]interface{},0,10)
        p = append(p,"c")
        if _,err := sc.Invoke_method("delete_entry",p); err == nil {
            fmt.Printf("Deleted 'c'\n")
        } else {
            fmt.Printf("Error invoking delete_entry: %v\n",err)
            os.Exit(1)
        }

        fmt.Printf("Delete 'a'.\n")
        p = make([]interface{},0,10)
        p = append(p,"a")
        if _,err := sc.Invoke_method("delete_entry",p); err == nil {
            fmt.Printf("Deleted 'a'\n")
        } else {
            fmt.Printf("Error invoking delete_entry: %v\n",err)
            os.Exit(1)
        }

        fmt.Printf("Retrieve a list of all registered names, should be empty.\n")
        p = make([]interface{},0,10)
        p = append(p,0)
        p = append(p,1)
        if nl,err := sc.Invoke_method("get_names",p); err == nil {
            fmt.Printf("Registered names %v\n",nl)
        } else {
            fmt.Printf("Error invoking get_names: %v\n",err)
            os.Exit(1)
        }

    } else {
        fmt.Printf("Deployment error %v\n",err)
    }

    fmt.Println("Terminating client")
}
