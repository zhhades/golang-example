package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"os"
)

type GPS struct {
	Longitude float64 `json:"longitude" gorm:"type:varchar(20) column:longitude"`
	Latitude  float64 `json:"latitude" gorm:"type:varchar(20) column:latitude"`
	GpsTime   int64   `json:"gpsTime" gorm:"type:bigint column:gps_time"`
}

func (g GPS) TableName() string {
	return "tb_gps_log"
}

func ReadFile(fileName string) []string {
	var (
		res  []string
		file *os.File
		err  error
	)
	defer file.Close()
	file, err = os.Open(fileName)
	if err != nil {
		log.Printf("读取文件失败")
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		res = append(res, scanner.Text())
	}
	return res
}

func InitDB() *gorm.DB {
	username := "root"        //账号
	password := "5zgmvzXY3fY" //密码
	host := "10.122.100.146"  //数据库地址，可以是Ip或者域名
	port := 3306              //数据库端口
	Dbname := "test"          //数据库名
	timeout := "100s"

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local&timeout=%s", username, password, host, port, Dbname, timeout)
	//连接MYSQL, 获得DB类型实例，用于后面的数据库读写操作。
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("连接数据库失败, error=" + err.Error())
	}
	return db
}

func main() {
	var (
		gpsArray []GPS
		gps      GPS
	)
	db := InitDB()
	for _, str := range ReadFile("handler_gps_log/gps.log") {
		json.Unmarshal([]byte(str), &gps)

		gpsArray = append(gpsArray, gps)
	}

	batches := db.CreateInBatches(gpsArray, len(gpsArray))
	fmt.Println(batches)
}
