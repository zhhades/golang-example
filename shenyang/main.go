package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type ReqBody struct {
	Data      string `json:"data"`
	Sign      string `json:"sign"`
	Timestamp int64  `json:"timestamp"`
}

type MonitorStartParam struct {
	CameraType string `json:"cameraType"`
	StreamMode int    `json:"streamMode"`
}

func SHA1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func sign(reqBody string, time int64, signatureKey string) string {
	str := base64.StdEncoding.EncodeToString([]byte(reqBody)) + strconv.FormatInt(time, 10) + signatureKey

	return SHA1(str)
}

func main() {
	unixTime := time.Now().UnixNano() / 1e6
	startURL := "https://portal.anddrive.cn/api/v0/thirdpart/terminals/860404040004262/monitor-start"
	continueURL := "https://portal.anddrive.cn/api/v0/thirdpart/terminals/860404040004262/monitor-continue"
	headMap := map[string]string{
		"Content-Type": "application/json;charset=UTF-8",
		"sysCode":      "30007",
		"from":         "OTHER",
		"msgId":        strconv.FormatInt(unixTime, 10),
	}

	MonitorStart(unixTime, startURL, headMap)
	for {
		unixTime = time.Now().UnixNano() / 1e6
		headMap["msgId"] = strconv.FormatInt(unixTime, 10)
		MonitorContinue(unixTime, continueURL, headMap)
		time.Sleep(1 * time.Minute)
	}

}

func MonitorContinue(unixTime int64, continueURL string, headMap map[string]string) {
	param := make(map[string]interface{})
	paramJson, err := json.Marshal(param)
	signStr := sign(string(paramJson), unixTime, "I51#H!44uR3")
	reqBody := ReqBody{
		Data:      base64.StdEncoding.EncodeToString(paramJson),
		Sign:      signStr,
		Timestamp: unixTime,
	}
	reqBodyJson, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", continueURL, bytes.NewBuffer(reqBodyJson))
	for k, v := range headMap {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Body:", string(body))

}

func MonitorStart(unixTime int64, startURL string, headMap map[string]string) {
	param := MonitorStartParam{
		CameraType: "FRONT",
		StreamMode: 0,
	}
	paramJson, err := json.Marshal(param)
	if err != nil {
		fmt.Println("error is " + err.Error())
	}
	fmt.Println(string(paramJson))
	signStr := sign(string(paramJson), unixTime, "I51#H!44uR3")

	reqBody := ReqBody{
		Data:      base64.StdEncoding.EncodeToString(paramJson),
		Sign:      signStr,
		Timestamp: unixTime,
	}
	reqBodyJson, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", startURL, bytes.NewBuffer(reqBodyJson))
	for k, v := range headMap {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
}
