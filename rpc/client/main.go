package main

import (
	"encoding/json"
	"fmt"
	"github.com/kirinlabs/HttpRequest"
)

type ResponseData struct {
	Data int `json:"data"`
}

func Add(a, b int) int {
	req := HttpRequest.NewRequest()
	res, _ := req.Get(fmt.Sprintf("http://localhost:8000/%s?a=%d&b=%d", "add", a, b))
	body, _ := res.Body()
	resData := &ResponseData{}
	json.Unmarshal(body, resData)
	return resData.Data
}

func main() {
	fmt.Println(Add(1, 2))
}
