package main

import (
	"fmt"
	"net/rpc"
)

func main() {
	//1.建立连接
	cli, _ := rpc.Dial("tcp", "localhost:1103")
	var reply string
	err := cli.Call("HelloService.Hello", "zhhades", &reply)
	if err != nil {
		panic(err)
	}
	fmt.Println(reply)
}
