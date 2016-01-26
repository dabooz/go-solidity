package main 

import (
    "fmt"
    "repo.hovitos.engineering/MTN/go-solidity/contract_api"
    "io/ioutil"
    "os"
    "strconv"
)

func main() {
    fmt.Println("Starting MTN contract bootstrap.")

    if len(os.Args) < 2 {
        fmt.Printf("Need more than %v parameters.",len(os.Args))
        os.Exit(1)
    }
    owning_acount := os.Args[1]

    fmt.Printf("Using account %v.\n",owning_acount)

    dir_ver := Get_directory_version()
    fmt.Printf("Bootstrapping version %v entries.\n",dir_ver)

    fmt.Println("Deploying directory contract.")
    dsc := contract_api.SolidityContractFactory("directory")
    if res,err := dsc.Deploy_contract(owning_acount, ""); err == nil {
        fmt.Printf("Deployed directory contract:%v at %v.\n",res,dsc.Get_contract_address())

        fmt.Println("Deploying token bank contract.")
        tbsc := contract_api.SolidityContractFactory("token_bank")
        if res,err := tbsc.Deploy_contract(owning_acount, ""); err == nil {
            fmt.Printf("Deployed token_bank contract:%v at %v.\n",res,tbsc.Get_contract_address())

            fmt.Println("Deploying device_registry contract.")
            drsc := contract_api.SolidityContractFactory("device_registry")
            if res,err := drsc.Deploy_contract(owning_acount, ""); err == nil {
                fmt.Printf("Deployed device_registry contract:%v at %v.\n",res,drsc.Get_contract_address())

                // Connect contracts together

                fmt.Println("Adding token bank to directory.")
                p := make([]interface{},0,10)
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

                // Saving directory address to file system

                _ = ioutil.WriteFile("directory",[]byte(dsc.Get_contract_address()[2:]),0644)
                fmt.Printf("Wrote directory address to file system.\n")

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

                fmt.Println("Successfully completed MTN contract bootstrap.")

            } else {
                fmt.Printf("Error deploying device_registry: %v\n",err)
                os.Exit(1)
            }

        } else {
            fmt.Printf("Error deploying token_bank: %v\n",err)
            os.Exit(1)
        }

    } else {
        fmt.Printf("Error deploying directory: %v\n",err)
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


