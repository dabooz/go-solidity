package main 

import (
    "fmt"
    "github.com/open-horizon/go-solidity/contract_api"
    "io/ioutil"
    "os"
    "strconv"
    "strings"
)

func main() {
    fmt.Println("Starting Horizon V2 contract bootstrap.")

    if len(os.Args) < 2 {
        fmt.Printf("Need more than %v parameters.",len(os.Args))
        os.Exit(1)
    }
    owning_acount := os.Args[1]
    var existing_dir = ""
    if len(os.Args)>2 {
        existing_dir = os.Args[2]  // This parameter is optional
        fmt.Printf("Bootstrap into existing dir %v.\n",existing_dir)
    }

    if !strings.HasPrefix(owning_acount, "0x") {
        owning_acount = "0x" + owning_acount
    }

    if len(existing_dir) != 0 && !strings.HasPrefix(existing_dir, "0x") {
        existing_dir = "0x" + existing_dir
    }

    fmt.Printf("Using account %v.\n",owning_acount)
    
    dir_ver := Get_directory_version()
    if dir_ver != 0 {
        fmt.Printf("Bootstrap version %v into directory.\n",dir_ver)
    }

    dsc := contract_api.SolidityContractFactory("directory")
    if existing_dir == "" {
        fmt.Println("Deploying directory contract.")
        if res,err := dsc.Deploy_contract(owning_acount, ""); err == nil {
            fmt.Printf("Deployed directory contract:%v at %v.\n",res,dsc.Get_contract_address())
        } else {
            fmt.Printf("Error deploying directory: %v\n",err)
            os.Exit(1)
        }
    } else {
        fmt.Println("Locating directory contract.")
        if res,err := dsc.Load_contract(owning_acount, ""); err == nil {
            dsc.Set_contract_address(existing_dir)
            fmt.Printf("Located directory contract:%v at %v.\n",res,dsc.Get_contract_address())
        } else {
            fmt.Printf("Error deploying directory: %v\n",err)
            os.Exit(1)
        }
    }

    fmt.Println("Deploying agreements contract.")
    agsc := contract_api.SolidityContractFactory("agreements")
    if res,err := agsc.Deploy_contract(owning_acount, ""); err == nil {
        fmt.Printf("Deployed agreements contract:%v at %v.\n",res,agsc.Get_contract_address())
    } else {
        fmt.Printf("Error deploying agreements: %v\n",err)
        os.Exit(1)
    }

    // Connect contracts together

    fmt.Println("Adding agreements to directory.")
    p := make([]interface{},0,10)
    p = append(p,"agreements")
    p = append(p,agsc.Get_contract_address())
    p = append(p,dir_ver)
    if _, err := dsc.Invoke_method("add_entry",p); err != nil {
        fmt.Printf("Error adding agreements to directory: %v\n",err)
        os.Exit(1)
    } else {
        fmt.Println("Added agreements to directory.")
    }

    // Saving directory address to file system

    if dir_ver == 0 {
        con_addr := dsc.Get_contract_address()
        if strings.HasPrefix(con_addr,"0x") {
            _ = ioutil.WriteFile("directory",[]byte(con_addr[2:]),0644)
        } else {
            _ = ioutil.WriteFile("directory",[]byte(con_addr),0644)
        }
        fmt.Printf("Wrote directory address to file system.\n")
    }

    fmt.Println("Successfully completed Horizon V2 contract bootstrap.")


}

func Get_directory_version() int {
    dir_ver := os.Getenv("CMTN_DIRECTORY_VERSION")
    if dir_ver == "" {
        dir_ver = "0"
    }
    var err error
    var d_ver = 0;
    if d_ver,err = strconv.Atoi(dir_ver); err != nil {
        return 0
    }
    return d_ver
}


