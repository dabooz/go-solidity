package main

import (
    "bytes"
    "errors"
    "flag"
    "github.com/golang/glog"
    "io/ioutil"
    "net/http"
    "repo.hovitos.engineering/MTN/go-eth-rpc"
    "repo.hovitos.engineering/MTN/go-solidity/contract_api"
)

func main() {

    // ================== Initialization =============================================
    
    input_dirAddress := flag.String("dirAddr", "", "Directory contract address")
    input_version := flag.Int("version", 0, "Contract version to track")
    input_contract := flag.String("contract", "", "Contract address to deregister")

    flag.Parse()

    glog.Infof("Starting Deregister Utility")

    glog.Infof("Input directory address: %v", *input_dirAddress)
    dirAddress := ""
    if *input_dirAddress != "" {
        dirAddress = *input_dirAddress
    } else {
        if dirAddress = getDirectoryAddress(); dirAddress == "" {
            panic(errors.New("No directory address"))
        }
    }
    glog.Infof("Using directory address: %v", dirAddress)

    glog.Infof("Input version: %v", *input_version)
    version := *input_version
    glog.Infof("Working with contracts from version %v", version)

    ethAccount := ""
    rpcc := getRPCClient()
    if rpcc == nil {
        panic(errors.New("No ethereum account has been created"))
    } else {
        ethAccount,_ = rpcc.Get_first_account()
    }
    glog.Infof("Running under account: %v", ethAccount)

    glog.Infof("Input contract: %v", *input_contract)
    contractAddress := ""
    if *input_contract != "" {
        contractAddress = *input_contract
    } else {
        panic(errors.New("No contract address"))
    }
    glog.Infof("Working with contract %v", contractAddress)

    // ================== Mainline =====================================================

    // Find the address of the other contracts from the directory
    dir := contract_api.SolidityContractFactory("directory")
    dir.Set_skip_eventlistener()
    dir.Set_contract_address(dirAddress)
    if _,err := dir.Load_contract(ethAccount, ""); err != nil {
        glog.Errorf("Debug: Error loading directory contract: %v\n",err)
        panic(err)
    }

    // Find the address of the device registry
    if deviceRegistryAddress, err := getContractAddress(dir, "device_registry", version); err != nil {
        glog.Errorf("Debug: Error finding device registry contract address: %v\n",err)
        panic(err)
    } else {
        // Look for the target contract address within the device registry
        // First load the solidity contract definition for the device resgitry.
        dr := contract_api.SolidityContractFactory("device_registry")
        dr.Set_skip_eventlistener()
        dr.Set_contract_address(deviceRegistryAddress)
        if _,err := dr.Load_contract(ethAccount, ""); err != nil {
            glog.Errorf("Debug: Error loading device resgistry contract: %v\n",err)
            panic(err)
        }

        // Then lookup the target contract address in the device registry
        p := make([]interface{},0,10)
        p = append(p,contractAddress)
        if desc,err := dr.Invoke_method("get_description",p); err != nil {
            glog.Errorf("Debug: Error looking for %v in device registry: %v\n", contractAddress, err)
            panic(err)
        } else {
            switch desc.(type) {
                case interface{}:
                    var array_attrib []string
                    array_attrib = desc.([]string)
                    if len(array_attrib) > 0 {
                        glog.Infof("Deleting contract with attributes: %v", array_attrib)
                        p := make([]interface{},0,10)
                        p = append(p,contractAddress)
                        if desc,err := dr.Invoke_method("deregister",p); err != nil {
                            glog.Errorf("Debug: Error deregistering %v in device registry: %v\n", contractAddress, err)
                            panic(err)
                        } else {
                            log.Infof("Device deregistered.")
                        }
                    }
                default:
                    glog.Errorf("Description is not interface: %v", desc)
            }
        }
    }

}

func getDirectoryAddress() string {
    url := "https://dal05.objectstorage.softlayer.net/v1/AUTH_773b8ed6-b3c8-4683-9d7a-dbe2ee11095e/volcano/directory.address"
    if outBytes, err := invoke_HTTP("GET", url, nil); err != nil {
        return ""
    } else {
        return string(outBytes)
    }
}

func getContractAddress(dir *contract_api.SolidityContract, contract string, version int) (string, error) {
    p := make([]interface{},0,10)
    p = append(p,contract)
    p = append(p,version)
    if draddr,err := dir.Invoke_method("get_entry_by_version",p); err != nil {
        glog.Errorf("Debug: Could not find %v in directory: %v\n", contract, err)
        return "", err
    } else {
        return draddr.(string), nil
    }
}

func getRPCClient() *go_eth_rpc.RPC_Client {
    var rpcc *go_eth_rpc.RPC_Client

    if con := go_eth_rpc.RPC_Connection_Factory("", 0, "http://localhost:8545"); con == nil {
        glog.Errorf("RPC Connection not created")
        return nil
    } else if rpcc = go_eth_rpc.RPC_Client_Factory(con); rpcc == nil {
        glog.Errorf("RPC Client not created")
        return nil
    }
    return rpcc
}

func invoke_HTTP(method string, url string, body []byte) ([]byte,error) {
    var out []byte

    if req, err := http.NewRequest(method, url, bytes.NewBuffer(body)); err != nil {
        glog.Errorf("Unable to create request for %v %v, error: %v", method, url, err)
        return out, err
    } else {
        req.Close = true            // work around to ensure that Go doesn't get connections confused. Supposed to be fixed in Go 1.6.
        client := &http.Client{}
        rawresp, err := client.Do(req)
        if err != nil {
            glog.Errorf("Error invoking %v %v, error: %v", method, url, err)
            return out, err
        }
        defer rawresp.Body.Close()
        glog.Infof("response Status:", rawresp.Status)
        if out, err := ioutil.ReadAll(rawresp.Body); err != nil {
            glog.Errorf("Error reading response to %v %v, error: %v", method, url, err)
            return out, err
        } else {
            return out, nil
        }
    }
}
