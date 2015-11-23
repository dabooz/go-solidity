package main

import (
    "fmt"
    "repo.hovitos.engineering/MTN/go-solidity/contract_api"
    "os"
    )

func main() {
    fmt.Println("Starting directory client")

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
    } else {
        fmt.Printf("Error invoking get_entry: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Retrieve a list of all registered names, should have only the MTN platform entries.\n")
    p = make([]interface{},0,10)
    p = append(p,0)
    p = append(p,1)
    if nl,err := dirc.Invoke_method("get_names",p); err == nil {
        fmt.Printf("Registered names %v\n",nl)
    } else {
        fmt.Printf("Error invoking get_names: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Register 'a' with address 0x0000000000000000000000000000000000000010.\n")
    p = make([]interface{},0,10)
    p = append(p,"a")
    p = append(p,"0x0000000000000000000000000000000000000010")
    if _,err := dirc.Invoke_method("add_entry",p); err == nil {
        fmt.Printf("Registered 'a'.\n")
    } else {
        fmt.Printf("Error invoking add_entry: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Retrieve 'a', should have address 10.\n")
    p = make([]interface{},0,10)
    p = append(p,"a")
    if aa,err := dirc.Invoke_method("get_entry",p); err == nil {
        fmt.Printf("Retrieved 'a', is %v.\n",aa)
    } else {
        fmt.Printf("Error invoking add_entry: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Retrieve owner of 'a', should be %v.\n",registry_owner)
    p = make([]interface{},0,10)
    p = append(p,"a")
    if aa,err := dirc.Invoke_method("get_entry_owner",p); err == nil {
        fmt.Printf("Retrieved owner of 'a' %v.\n",aa)
    } else {
        fmt.Printf("Error invoking add_entry: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Retrieve a list of all registered names, should have 'a' in it.\n")
    p = make([]interface{},0,10)
    p = append(p,0)
    p = append(p,1)
    if nl,err := dirc.Invoke_method("get_names",p); err == nil {
        fmt.Printf("Registered names %v\n",nl)
    } else {
        fmt.Printf("Error invoking get_names: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Register 'b' with address 0x0000000000000000000000000000000000000011.\n")
    p = make([]interface{},0,11)
    p = append(p,"b")
    p = append(p,"0x0000000000000000000000000000000000000011")
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
    if _,err := dirc.Invoke_method("add_entry",p); err == nil {
        fmt.Printf("Registered 'c'.\n")
    } else {
        fmt.Printf("Error invoking add_entry: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Retrieve a list of all registered names, should have 'a,b,c' in it.\n")
    p = make([]interface{},0,10)
    p = append(p,0)
    p = append(p,2)
    if nl,err := dirc.Invoke_method("get_names",p); err == nil {
        fmt.Printf("Registered names %v\n",nl)
    } else {
        fmt.Printf("Error invoking get_names: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Delete 'b'.\n")
    p = make([]interface{},0,10)
    p = append(p,"b")
    if _,err := dirc.Invoke_method("delete_entry",p); err == nil {
        fmt.Printf("Deleted 'b'\n")
    } else {
        fmt.Printf("Error invoking delete_entry: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Retrieve a list of all registered names, should have 'a,c' in it.\n")
    p = make([]interface{},0,10)
    p = append(p,0)
    p = append(p,2)
    if nl,err := dirc.Invoke_method("get_names",p); err == nil {
        fmt.Printf("Registered names %v\n",nl)
    } else {
        fmt.Printf("Error invoking get_names: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Delete 'c'.\n")
    p = make([]interface{},0,10)
    p = append(p,"c")
    if _,err := dirc.Invoke_method("delete_entry",p); err == nil {
        fmt.Printf("Deleted 'c'\n")
    } else {
        fmt.Printf("Error invoking delete_entry: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Delete 'a'.\n")
    p = make([]interface{},0,10)
    p = append(p,"a")
    if _,err := dirc.Invoke_method("delete_entry",p); err == nil {
        fmt.Printf("Deleted 'a'\n")
    } else {
        fmt.Printf("Error invoking delete_entry: %v\n",err)
        os.Exit(1)
    }

    fmt.Printf("Retrieve a list of all registered names, should be just the MTN platform entries.\n")
    p = make([]interface{},0,10)
    p = append(p,0)
    p = append(p,1)
    if nl,err := dirc.Invoke_method("get_names",p); err == nil {
        fmt.Printf("Registered names %v\n",nl)
    } else {
        fmt.Printf("Error invoking get_names: %v\n",err)
        os.Exit(1)
    }

    fmt.Println("Terminating directory test client")
}
