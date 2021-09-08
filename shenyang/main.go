package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/suifengtec/gocoord"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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

type GPS struct {
	Longitude   float64 `json:"longitude" gorm:"type:varchar(20) column:longitude"`
	LongitudeBD float64 `gorm:"type:varchar(20) column:longitudeBD"`
	Latitude    float64 `json:"latitude" gorm:"type:varchar(20) column:latitude"`
	LatitudeBD  float64 `gorm:"type:varchar(20) column:latitudeBD"`
	GpsTime     int64   `json:"gpsTime" gorm:"type:bigint column:gps_time"`
}

func (g GPS) TableName() string {
	return "tb_gps_log"
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

var _db *gorm.DB

func init() {
	username := "root"
	password := "5zgmvzXY3fY"
	host := "10.122.100.146"
	port := 3306
	Dbname := "test"
	timeout := "100s"

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local&timeout=%s", username, password, host, port, Dbname, timeout)
	var err error
	_db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("连接数据库失败, error=" + err.Error())
	}
	sqlDB, _ := _db.DB()
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(20)
}

func GetDB() *gorm.DB {
	return _db
}

func main() {
	unixTime := time.Now().UnixNano() / 1e6
	startURL := "https://portal.anddrive.cn/api/v0/thirdpart/terminals/860404040006697/monitor-start"
	continueURL := "https://portal.anddrive.cn/api/v0/thirdpart/terminals/860404040006697/monitor-continue"
	LocationURL := "https://portal.anddrive.cn/api/v0/thirdpart/terminals/860404040006697/realtime?timestamp=%d&sign=%s"
	headMap := map[string]string{
		"Content-Type": "application/json;charset=UTF-8",
		"sysCode":      "30007",
		"from":         "OTHER",
		"msgId":        strconv.FormatInt(unixTime, 10),
	}

	MonitorStart(unixTime, startURL, headMap)

	go func() {
		for {
			GetLocation(unixTime, LocationURL)
			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		for {
			unixTime = time.Now().UnixNano() / 1e6
			headMap["msgId"] = strconv.FormatInt(unixTime, 10)
			MonitorContinue(unixTime, continueURL, headMap)
			time.Sleep(1 * time.Minute)
		}
	}()

	time.Sleep(time.Hour * 24)

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
	fmt.Println("续期 Status:", resp.Status)

}

func GetLocation(unixTime int64, url string) {
	headMap := map[string]string{
		"sysCode": "30007",
		"from":    "OTHER",
		"msgId":   strconv.FormatInt(unixTime, 10),
	}
	signStr := sign("", unixTime, "I51#H!44uR3")
	req, _ := http.NewRequest("GET", fmt.Sprintf(url, unixTime, signStr), nil)
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
	gps := GPS{}
	json.Unmarshal(body, &gps)
	p := gocoord.Position{Lon: gps.Longitude, Lat: gps.Latitude}
	bd09 := gocoord.WGS84ToBD09(p)
	gps.LongitudeBD = bd09.Lon
	gps.LatitudeBD = bd09.Lat
	db := GetDB()
	db.Create(&gps)
	fmt.Println("gps Status:", resp.Status)
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
	fmt.Println("开流 Body:", string(body))
}
