package main

import (
	"net"
	"net/rpc"
)

type HelloService struct {
}

func (s *HelloService) Hello(request string, reply *string) error {
	*reply = "hello, " + request
	return nil
}

func main() {
	listener, _ := net.Listen("tcp", ":1103")
	_ = rpc.RegisterName("HelloService", &HelloService{})
	conn, _ := listener.Accept()
	rpc.ServeConn(conn)
}
