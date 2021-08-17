package main

import (
	"encoding/json"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io/ioutil"
	"time"
)

type Device struct {
	CommandType string
	Item        []DeviceInfo
}

type DeviceInfo struct {
	IsCatalog  uint8  `json:"IsCatalog" gorm:"-"`
	Name       string `json:"Name" gorm:"name"`
	GbDeviceId string `json:"SubDeviceID" gorm:"gb_device_id"`
	Latitude   string `json:"Latitude" gorm:"latitude"`
	Longitude  string `json:"Longitude" gorm:"longitude"`
}

func FilterSlice(s []DeviceInfo) []DeviceInfo {
	newS := s[:0]
	for _, x := range s {
		if x.IsCatalog == 0 {
			newS = append(newS, x)
		}
	}
	return newS
}

func main() {
	dsn := "root:ZIWio9noSmo@tcp(10.172.198.44:3306)/shenyang?charset=utf8mb4&parseTime=True&loc=Local"
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	filepath := "D:\\07_GolandProjects\\project\\go-web-study\\go_rm_test\\devicelist20210701.txt"
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		panic(err.Error)
	}
	device := &Device{}
	err = json.Unmarshal(content, device)
	if err != nil {
		panic(err)
	}

	deviceInfos := FilterSlice(device.Item)
	now := time.Now()
	fmt.Println(now)

	db.Table("test_device").CreateInBatches(deviceInfos, 5000)

	db.Commit()

	fmt.Println(time.Since(now))

}