package contract_api

import (
    "encoding/json"
    "reflect"
    "strings"
    "testing"
    )

const testCCJSONString = `{"info": {"language": "Solidity","languageVersion": "0",`+
    `"abiDefinition": `+
        `[{"inputs": [],"type": "function","constant": true, "name": "get_owner", "outputs": [{"type": "address", "name": "r"}]},`+
        `{"inputs": [],"type": "function","constant": true, "name": "has_owner", "outputs": [{"type": "bool", "name": "r"}]},`+
        `{"inputs": [{"type": "uint256", "name": "_id"}, {"type": "string", "name": "_name"}, {"type": "string", "name": "_sha1"}], "type": "function","constant": false, "name": "new_container", "outputs": [{"type": "bool", "name": "r"}]},`+
        `{"inputs": [],"type": "function", "constant": false, "name": "kill", "outputs": []},`+
        `{"inputs": [], "type": "function", "constant": true, "name": "get_container_provider", "outputs": [{"type": "address", "name": "r"}]},`+
        `{"inputs": [],"type": "function", "constant": true, "name": "get_container_id", "outputs": [{"type": "uint256", "name": "r"}]},`+
        `{"inputs": [], "type": "function", "constant": false, "name": "exec_complete", "outputs": [{"type": "bool", "name": "r"}]},`+
        `{"inputs": [], "type": "function", "constant": true, "name": "get_container_name", "outputs": [{"type": "string", "name": "r"}]},`+
        `{"inputs": [],"type": "function", "constant": true, "name": "get_sha1", "outputs": [{"type": "string", "name": "r"}]},`+
        `{"inputs": [{"type": "address", "name": "_bank"}],"type": "function", "constant": false, "name": "set_bank", "outputs": []},`+
        `{"inputs": [{"type": "address", "name": "_bank"},{"type": "uint256[]", "name": "_attrib"}],"type": "function", "constant": false, "name": "set_attributes", "outputs": [{"type": "uint256[]", "name": "_attrib"}]},`+
        `{"inputs": [], "type": "constructor"},`+
        `{"inputs": [{"indexed": true, "type": "uint256", "name": "_eventcode"}, {"indexed": false, "type": "uint256", "name": "_id"}],"type": "event", "name": "NewContainer", "anonymous": false},{"inputs": [{"indexed": true, "type": "uint256", "name": "_eventcode"}, {"indexed": false, "type": "uint256", "name": "_id"}, {"indexed": true, "type": "address", "name": "_self"}],"type": "event", "name": "ExecutionComplete", "anonymous": false}],`+
        `"compilerVersion": "0.1.1","developerDoc": {"methods": {}}, "userDoc": {"methods": {}}}}`


func TestConstructor(t *testing.T) {
    sc := SolidityContractFactory("some_contract")
    if sc == nil {
        t.Errorf("Factory returned nil, but should not.\n")
    }
}

func TestContractAddress(t *testing.T) {
    sc := SolidityContractFactory("some_contract")
    ca := "0x0123456789"
    sc.Set_contract_address(ca)
    if sc.Get_contract_address() != ca {
        t.Errorf("ContractAddress setter failed, expected %v, received %v\n",ca,sc.Get_contract_address())
    }
}

func TestFunctionSearch(t *testing.T) {
    sc := SolidityContractFactory("some_contract")
    err := json.Unmarshal([]byte(testCCJSONString),&sc.compiledContract)
    if err == nil {
        functionDef := sc.getFunctionFromABI("get_owner")
        if functionDef == nil {
            t.Errorf("ABI Search unable to find function get_owner.\n")
        }
    } else {
        t.Errorf("Error Unmarshalling test JSON, error: %v\n",err)
    }
}

func TestNegFunctionSearch(t *testing.T) {
    sc := SolidityContractFactory("some_contract")
    err := json.Unmarshal([]byte(testCCJSONString),&sc.compiledContract)
    if err == nil {
        functionDef := sc.getFunctionFromABI("foobar")
        if functionDef != nil {
            t.Errorf("ABI Search found non-existent foobar method.\n")
        }
    } else {
        t.Errorf("Error Unmarshalling test JSON, error: %v\n",err)
    }
}

func TestZeroPad(t *testing.T) {
    sc := SolidityContractFactory("some_contract")
    res := ""
    expected := ""

    expected = "0000000000000000000000000000000000000000000000000000000000001111"
    res = sc.zero_pad_left("1111", 64)
    if len(res) != len(expected) || res != expected {
        t.Errorf("zero_pad_left returned %v, expected %v\n",res,expected)
    }

    expected = "000000000000000000000000000000000000000000000000000000001111"
    res = sc.zero_pad_left("1111", 60)
    if len(res) != len(expected) || res != expected {
        t.Errorf("zero_pad_left returned %v, expected %v\n",res,expected)
    }

    expected = "1111"
    res = sc.zero_pad_left("1111", 4)
    if len(res) != len(expected) || res != expected {
        t.Errorf("zero_pad_left returned %v, expected %v\n",res,expected)
    }

    expected = "00000001234567890123"
    res = sc.zero_pad_left("1234567890123", 10)
    if len(res) != len(expected) || res != expected {
        t.Errorf("zero_pad_left returned %v, expected %v\n",res,expected)
    }

}

func TestUInt256(t *testing.T) {
    sc := SolidityContractFactory("some_contract")
    res := ""
    err := error(nil)
    expected := ""

    expected = "000000000000000000000000000000000000000000000000000000000000000a"
    res,err = sc.encode_uint256("some_method", 10)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encode_uint256 returned %v, expected %v. Error:%v\n",res,expected,err)
    }

    expected = "000000000000000000000000000000000000000000000000112210f47de98115"
    res,err = sc.encode_uint256("some_method", 1234567890123456789)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encode_uint256 returned %v, expected %v. Error:%v\n",res,expected,err)
    }

    res,err = sc.encode_uint256("some_method", "jkl")
    if err == nil {
        t.Errorf("encode_uint256 returned %v, expected %v.\n",res,"error")
    } else {
        switch err.(type) {
            case *UnsupportedValueError:
                // this is expected
            default:
                t.Errorf("encode_uint256 returned %v, expected %v.\n",reflect.TypeOf(err).String(),"UnsupportedValueError")
        }
    }

    res,err = sc.encode_uint256("some_method", nil)
    if err == nil {
        t.Errorf("encode_uint256 returned %v, expected %v.\n",res,"error")
    } else {
        switch err.(type) {
            case *UnsupportedTypeError:
                // this is expected
            default:
                t.Errorf("encode_uint256 returned %v, expected %v.\n",reflect.TypeOf(err).String(),"UnsupportedTypeError")
        }
    }

    res,err = sc.encode_uint256("some_method", 10.5)
    if err == nil {
        t.Errorf("encode_uint256 returned %v, expected %v.\n",res,"error")
    } else {
        switch err.(type) {
            case *UnsupportedTypeError:
                // this is expected
            default:
                t.Errorf("encode_uint256 returned %v, expected %v.\n",reflect.TypeOf(err).String(),"UnsupportedTypeError")
        }
    }

    res,err = sc.encode_uint256("some_method", -10)
    if err == nil {
        t.Errorf("encode_uint256 returned %v, expected %v.\n",res,"error")
    } else {
        switch err.(type) {
            case *UnsupportedValueError:
                // this is expected
            default:
                t.Errorf("encode_uint256 returned %v, expected %v.\n",reflect.TypeOf(err).String(),"UnsupportedValueError")
        }
    }
}

func TestUInt256array(t *testing.T) {
    sc := SolidityContractFactory("some_contract")
    res := ""
    err := error(nil)
    input := make([]int, 0, 10)
    expected := ""

    expected = "0000000000000000000000000000000000000000000000000000000000000000"
    res,err = sc.encode_uint256_array("some_method", input)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encode_uint256_array returned %v, expected %v. Error:%v\n",res,expected,err)
    }

    input = make([]int, 0, 10)
    expected = "0000000000000000000000000000000000000000000000000000000000000002"
    expected += "000000000000000000000000000000000000000000000000000000000000000a"
    expected += "000000000000000000000000000000000000000000000000000000000000000b"
    input = append(input,10)
    input = append(input,11)
    res,err = sc.encode_uint256_array("some_method", input)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encode_uint256_array returned %v, expected %v. Error:%v\n",res,expected,err)
    }
    
    input = make([]int, 0, 10)
    input = append(input,10)
    input = append(input,-2)
    res,err = sc.encode_uint256_array("some_method", input)
    if err == nil {
        t.Errorf("encode_uint256_array returned %v, expected %v.\n",res,"error")
    } else {
        switch err.(type) {
            case *UnsupportedValueError:
                // this is expected
            default:
                t.Errorf("encode_uint256_array returned %v, expected %v.\n",reflect.TypeOf(err).String(),"UnsupportedValueError")
        }
    }

    res,err = sc.encode_uint256_array("some_method", nil)
    if err == nil {
        t.Errorf("encode_uint256_array returned %v, expected %v.\n",res,"error")
    } else {
        switch err.(type) {
            case *UnsupportedTypeError:
                // this is expected
            default:
                t.Errorf("encode_uint256_array returned %v, expected %v.\n",reflect.TypeOf(err).String(),"UnsupportedTypeError")
        }
    }

    inputs := make([]string, 0, 10)
    inputs = append(inputs,"10")
    inputs = append(inputs,"-2")
    res,err = sc.encode_uint256_array("some_method", inputs)
    if err == nil {
        t.Errorf("encode_uint256_array returned %v, expected %v.\n",res,"error")
    } else {
        switch err.(type) {
            case *UnsupportedTypeError:
                // this is expected
            default:
                t.Errorf("encode_uint256_array returned %v, expected %v.\n",reflect.TypeOf(err).String(),"UnsupportedTypeError")
        }
    }

}

func TestString(t *testing.T) {
    sc := SolidityContractFactory("some_contract")
    res := ""
    err := error(nil)
    expected := ""
    input := ""

    expected = "0000000000000000000000000000000000000000000000000000000000000000"
    expected += "0000000000000000000000000000000000000000000000000000000000000000"
    res,err = sc.encode_string("some_method", input)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encode_string returned %v, expected %v. Error:%v\n",res,expected,err)
    }

    input = "hello there"
    expected = "000000000000000000000000000000000000000000000000000000000000000b"
    expected += "68656c6c6f207468657265000000000000000000000000000000000000000000"
    res,err = sc.encode_string("some_method", input)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encode_string returned %v, expected %v. Error:%v\n",res,expected,err)
    }

    input = strings.Repeat("1234567890",7)
    expected = "0000000000000000000000000000000000000000000000000000000000000046"
    expected += "3132333435363738393031323334353637383930313233343536373839303132"
    expected += "3334353637383930313233343536373839303132333435363738393031323334"
    expected += "3536373839300000000000000000000000000000000000000000000000000000"
    res,err = sc.encode_string("some_method", input)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encode_string returned %v, expected %v. Error:%v\n",res,expected,err)
    }

    res,err = sc.encode_string("some_method", nil)
    if err == nil {
        t.Errorf("encode_string returned %v, expected %v.\n",res,"error")
    } else {
        switch err.(type) {
            case *UnsupportedTypeError:
                // this is expected
            default:
                t.Errorf("encode_string returned %v, expected %v.\n",reflect.TypeOf(err).String(),"UnsupportedTypeError")
        }
    }

    res,err = sc.encode_string("some_method", 1)
    if err == nil {
        t.Errorf("encode_string returned %v, expected %v.\n",res,"error")
    } else {
        switch err.(type) {
            case *UnsupportedTypeError:
                // this is expected
            default:
                t.Errorf("encode_string returned %v, expected %v.\n",reflect.TypeOf(err).String(),"UnsupportedTypeError")
        }
    }
}

func TestAddress(t *testing.T) {
    sc := SolidityContractFactory("some_contract")
    res := ""
    err := error(nil)
    expected := ""
    input := ""
    num := 0

    input = "0xb37e8570f16682474894d435b207bb9a67dec3d9"
    expected = "000000000000000000000000b37e8570f16682474894d435b207bb9a67dec3d9"
    res,err = sc.encode_address("some_method", input)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encode_address returned %v, expected %v. Error:%v\n",res,expected,err)
    }

    input = "b37e8570f16682474894d435b207bb9a67dec3d9"
    expected = "000000000000000000000000b37e8570f16682474894d435b207bb9a67dec3d9"
    res,err = sc.encode_address("some_method", input)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encode_address returned %v, expected %v. Error:%v\n",res,expected,err)
    }

    num = 1
    res,err = sc.encode_address("some_method", num)
    if err == nil {
        t.Errorf("encode_address returned %v, expected %v.\n",res,"error")
    } else {
        switch err.(type) {
            case *UnsupportedTypeError:
                // this is expected
            default:
                t.Errorf("encode_address returned %v, expected %v.\n",reflect.TypeOf(err).String(),"UnsupportedTypeError")
        }
    }

    input = ""
    res,err = sc.encode_address("some_method", input)
    if err == nil {
        t.Errorf("encode_address returned %v, expected %v.\n",res,"error")
    } else {
        switch err.(type) {
            case *UnsupportedValueError:
                // this is expected
            default:
                t.Errorf("encode_address returned %v, expected %v.\n",reflect.TypeOf(err).String(),"UnsupportedValueError")
        }
    }

    res,err = sc.encode_address("some_method", nil)
    if err == nil {
        t.Errorf("encode_address returned %v, expected %v.\n",res,"error")
    } else {
        switch err.(type) {
            case *UnsupportedTypeError:
                // this is expected
            default:
                t.Errorf("encode_address returned %v, expected %v.\n",reflect.TypeOf(err).String(),"UnsupportedTypeError")
        }
    }
}

func TestEncodeInputString(t *testing.T) {
    sc := SolidityContractFactory("some_contract")
    jsonerr := json.Unmarshal([]byte(testCCJSONString),&sc.compiledContract)
    if jsonerr != nil {
        t.Errorf("Error Unmarshalling test JSON, error: %v\n",jsonerr)
    }

    res := ""
    err := error(nil)
    expected := ""
    input := make([]interface{}, 0, 10)

    expected = ""
    res,err = sc.encodeInputString("get_owner", input)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encodeInputString returned %v, expected %v. Error:%v\n",res,expected,err)
    }

    expected = ""
    res,err = sc.encodeInputString("get_owner", nil)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encodeInputString returned %v, expected %v. Error:%v\n",res,expected,err)
    }

    expected = "0000000000000000000000000000000000000000000000000000000000000001"
    expected += "0000000000000000000000000000000000000000000000000000000000000060"
    expected += "00000000000000000000000000000000000000000000000000000000000000a0"
    expected += "0000000000000000000000000000000000000000000000000000000000000006"
    expected += "6c61746573740000000000000000000000000000000000000000000000000000"
    expected += "0000000000000000000000000000000000000000000000000000000000000004"
    expected += "6861736800000000000000000000000000000000000000000000000000000000"
    input = append(input,1)
    input = append(input,"latest")
    input = append(input,"hash")
    res,err = sc.encodeInputString("new_container", input)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encodeInputString returned %v, expected %v. Error:%v\n",res,expected,err)
    }

    expected = "0000000000000000000000000000000000000000000000000000000000000001"
    expected += "0000000000000000000000000000000000000000000000000000000000000060"
    expected += "00000000000000000000000000000000000000000000000000000000000000a0"
    expected += "0000000000000000000000000000000000000000000000000000000000000006"
    expected += "6c61746573740000000000000000000000000000000000000000000000000000"
    expected += "0000000000000000000000000000000000000000000000000000000000000004"
    expected += "6861736800000000000000000000000000000000000000000000000000000000"
    input = append(input,1)
    input = append(input,"latest")
    input = append(input,"hash")
    res,err = sc.encodeInputString("new_container", input)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encodeInputString returned %v, expected %v. Error:%v\n",res,expected,err)
    }

    expected = "000000000000000000000000b37e8570f16682474894d435b207bb9a67dec3d9"
    input = make([]interface{}, 0, 10)
    input = append(input,"0xb37e8570f16682474894d435b207bb9a67dec3d9")
    res,err = sc.encodeInputString("set_bank", input)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encodeInputString returned %v, expected %v. Error:%v\n",res,expected,err)
    }

    expected = "000000000000000000000000b37e8570f16682474894d435b207bb9a67dec3d9"
    expected += "0000000000000000000000000000000000000000000000000000000000000040"
    expected += "0000000000000000000000000000000000000000000000000000000000000002"
    expected += "0000000000000000000000000000000000000000000000000000000000000001"
    expected += "0000000000000000000000000000000000000000000000000000000000000fa0"
    input = make([]interface{}, 0, 10)
    input = append(input,"0xb37e8570f16682474894d435b207bb9a67dec3d9")
    in_array := make([]int, 0, 10)
    in_array = append(in_array,1)
    in_array = append(in_array,4000)
    input = append(input,in_array)
    res,err = sc.encodeInputString("set_attributes", input)
    if err != nil || len(res) != len(expected) || res != expected {
        t.Errorf("encodeInputString returned %v, expected %v. Error:%v\n",res,expected,err)
    }

    res,err = sc.encodeInputString("some_method", input)
    if err == nil {
        t.Errorf("encodeInputString returned %v, expected %v.\n",res,"error")
    } else {
        switch err.(type) {
            case *FunctionNotFoundError:
                // this is expected
            default:
                t.Errorf("encodeInputString returned %v, expected %v.\n",reflect.TypeOf(err).String(),"UnsupportedTypeError")
        }
    }

}

func TestDecodeOutputString(t *testing.T) {
    sc := SolidityContractFactory("some_contract")
    jsonerr := json.Unmarshal([]byte(testCCJSONString),&sc.compiledContract)
    if jsonerr != nil {
        t.Errorf("Error Unmarshalling test JSON, error: %v\n",jsonerr)
    }

    res := ""
    err := error(nil)
    expected := ""
    output := ""
    var out interface{}
    var expected_num,num uint64

    expected = "0xb37e8570f16682474894d435b207bb9a67dec3d9"
    output = "000000000000000000000000b37e8570f16682474894d435b207bb9a67dec3d9"
    out,err = sc.decodeOutputString("get_owner", output)
    if err == nil {
        switch out.(type) {
            case string:
                res = out.(string)
                if len(res) != len(expected) || res != expected {
                    t.Errorf("encodeOutputString returned %v, expected %v.\n",res,expected)
                }
            default:
                t.Errorf("encodeOutputString returned %v, expected %v.\n",reflect.TypeOf(out).String(),"string")
        }
    } else {
        t.Errorf("encodeOutputString returned %v, expected %v. Error:%v\n",out,expected,err)
    }

    expected_num = 10
    output = "000000000000000000000000000000000000000000000000000000000000000a"
    out,err = sc.decodeOutputString("get_container_id", output)
    if err == nil {
        switch out.(type) {
            case uint64:
                num = out.(uint64)
                if num != expected_num {
                    t.Errorf("encodeOutputString returned %v, expected %v.\n",num,expected_num)
                }
            default:
                t.Errorf("encodeOutputString returned %v, expected %v.\n",reflect.TypeOf(out).String(),"uint64")
        }
    } else {
        t.Errorf("encodeOutputString returned %v, expected %v. Error:%v\n",out,expected_num,err)
    }

    expected = "latest"
    output = "0000000000000000000000000000000000000000000000000000000000000020"
    output += "0000000000000000000000000000000000000000000000000000000000000006"
    output += "6c61746573740000000000000000000000000000000000000000000000000000"
    out,err = sc.decodeOutputString("get_container_name", output)
    if err == nil {
        switch out.(type) {
            case string:
                res = out.(string)
                if len(res) != len(expected) || res != expected {
                    t.Errorf("encodeOutputString returned %v, expected %v.\n",res,expected)
                }
            default:
                t.Errorf("encodeOutputString returned %v, expected %v.\n",reflect.TypeOf(out).String(),"string")
        }
    } else {
        t.Errorf("encodeOutputString returned %v, expected %v. Error:%v\n",out,expected,err)
    }

    expected_bool := true
    output = "0000000000000000000000000000000000000000000000000000000000000001"
    out,err = sc.decodeOutputString("has_owner", output)
    if err == nil {
        switch out.(type) {
            case bool:
                theBool := out.(bool)
                if theBool != expected_bool {
                    t.Errorf("encodeOutputString returned %v, expected %v.\n",theBool,expected_bool)
                }
            default:
                t.Errorf("encodeOutputString returned %v, expected %v.\n",reflect.TypeOf(out).String(),"bool")
        }
    } else {
        t.Errorf("encodeOutputString returned %v, expected %v. Error:%v\n",out,expected_bool,err)
    }

    output = "0000000000000000000000000000000000000000000000000000000000000020"
    output += "0000000000000000000000000000000000000000000000000000000000000002"
    output += "0000000000000000000000000000000000000000000000000000000000000001"
    output += "0000000000000000000000000000000000000000000000000000000000000fa0"
    expected_array := make([]uint64,0,10)
    expected_array = append(expected_array,1)
    expected_array = append(expected_array,4000)
    out,err = sc.decodeOutputString("set_attributes", output)
    if err == nil {
        switch out.(type) {
            case []uint64:
                theArr := out.([]uint64)
                if len(theArr) != len(expected_array) || !reflect.DeepEqual(theArr,expected_array) {
                    t.Errorf("encodeOutputString returned %v, expected %v.\n",theArr,expected_array)
                }
            default:
                t.Errorf("encodeOutputString returned %v, expected %v.\n",reflect.TypeOf(out).String(),"[]uint64")
        }
    } else {
        t.Errorf("encodeOutputString returned %v, expected %v. Error:%v\n",out,expected_array,err)
    }


}

func TestInvocationString(t *testing.T) {

    sc := SolidityContractFactory("some_contract")
    err := json.Unmarshal([]byte(testCCJSONString),&sc.compiledContract)
    if err == nil {
        //fmt.Printf("Unmarshalled string: %v\n",sc.compiledContract)
        //fmt.Printf("First function: %v\n",sc.compiledContract.Info.Abidefinition[0].Name)


    } else {
        t.Fatalf("Error Unmarshalling test JSON, error: %v\n",err)
    }


}



