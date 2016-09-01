package main 

import (
    "fmt"
    "repo.hovitos.engineering/MTN/go-solidity/contract_api"
    "io/ioutil"
    "os"
    "strconv"
    "strings"
)

func main() {
    fmt.Println("Starting MTN contract bootstrap.")

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

    fmt.Println("Deploying token bank contract.")
    tbsc := contract_api.SolidityContractFactory("token_bank")
    if res,err := tbsc.Deploy_contract(owning_acount, ""); err == nil {
        fmt.Printf("Deployed token_bank contract:%v at %v.\n",res,tbsc.Get_contract_address())

        fmt.Println("Deploying device_registry contract.")
        drsc := contract_api.SolidityContractFactory("device_registry")
        if res,err := drsc.Deploy_contract(owning_acount, ""); err == nil {
            fmt.Printf("Deployed device_registry contract:%v at %v.\n",res,drsc.Get_contract_address())

            // Connect contracts together

            fmt.Println("Adding agreements to directory.")
            p := make([]interface{},0,10)
            p = append(p,"agreements")
            p = append(p,agsc.Get_contract_address())
            p = append(p,dir_ver)
            _,_ = dsc.Invoke_method("add_entry",p)
            fmt.Println("Added agreements to directory.")

            fmt.Println("Adding token bank to directory.")
            p = make([]interface{},0,10)
            p = append(p,"token_bank")
            p = append(p,tbsc.Get_contract_address())
            p = append(p,dir_ver)
            _,_ = dsc.Invoke_method("add_entry",p)
            fmt.Println("Added token bank to directory.")

            fmt.Println("Adding device registry to directory.")
            p = make([]interface{},0,10)
            p = append(p,"device_registry")
            p = append(p,drsc.Get_contract_address())
            p = append(p,dir_ver)
            _,err = dsc.Invoke_method("add_entry",p)
            fmt.Println("Added device registry to directory.")

            fmt.Println("Connecting device registry to token bank.")
            p = make([]interface{},0,10)
            p = append(p,tbsc.Get_contract_address())
            _,err = drsc.Invoke_method("set_bank",p)
            fmt.Println("Connected device registry to token bank.")

            fmt.Println("Deploying whisper_directory contract.")
            wd := contract_api.SolidityContractFactory("whisper_directory")
            if res,err := wd.Deploy_contract(owning_acount, ""); err == nil {
                fmt.Printf("Deployed whisper_directory contract:%v at %v.\n",res,wd.Get_contract_address())
                fmt.Println("Adding whisper_directory to directory.")
                p := make([]interface{},0,10)
                p = append(p,"whisper_directory")
                p = append(p,wd.Get_contract_address())
                p = append(p,dir_ver)
                _,_ = dsc.Invoke_method("add_entry",p)
                fmt.Println("Added whisper_directory to directory.")
            } else {
                fmt.Printf("Error deploying whisper directory: %v\n",err)
                os.Exit(1)
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

            fmt.Println("Successfully completed MTN contract bootstrap.")

        } else {
            fmt.Printf("Error deploying device_registry: %v\n",err)
            os.Exit(1)
        }

    } else {
        fmt.Printf("Error deploying token_bank: %v\n",err)
        os.Exit(1)
    }

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


