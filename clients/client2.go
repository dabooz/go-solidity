package main

import (
    "fmt"
    "reflect"
    )

func main() {
    fmt.Println("Starting client1")

    genmap := map[string]interface{}{"cathy":2}

    intmap := map[string]int{"dave":1}

    x := genmap["cathy"]
    fmt.Printf("x is %v\n",x)
    fmt.Printf("x type %v\n",reflect.TypeOf(x).String())

    y := intmap["dave"]
    fmt.Printf("y is %v\n",y)

    // both fail
    intmap = map[string]int(genmap)
    genmap = map[string]interface{}(intmap)

    //genmap = intmap.(map[string]interface{})




    fmt.Println("Terminating client1")
}
