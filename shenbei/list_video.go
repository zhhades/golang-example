package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

type Video struct {
	Url    string `json:"url"`
	Status string `json:"status"`
	ID     string `json:"id"`
	Name   string `json:"name"`
}

type VideoCount struct {
	Data VideoCountData `json:"data"`
}

type VideoCountData struct {
	Total int `json:"totalRecords"`
}

type ListVideoRes struct {
	Videos []Video `json:"videos"`
}

func GetVideoCount(url string) int {

	headMap := map[string]string{
		"Authorization": "59dacfac729esuperadmin427f90bfa98c0a636e0c",
		"Content-Type":  "application/json;charset=UTF-8",
	}
	reqBody := map[string]interface{}{
		"action":   "all",
		"pageNo":   1,
		"pageSize": 1,
	}
	reqBodyJson, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBodyJson))
	for k, v := range headMap {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err.Error())
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	body, _ := ioutil.ReadAll(resp.Body)
	videoCount := &VideoCount{}
	if err := json.Unmarshal(body, videoCount); err != nil {
		fmt.Println(err.Error())
	}
	return videoCount.Data.Total
}

func GetVideos(wg *sync.WaitGroup, url string, listVideoResChan chan *ListVideoRes) {
	defer wg.Done()
	listVideoRes := &ListVideoRes{}
	DoGetVideos(url, listVideoRes)
	listVideoResChan <- listVideoRes
}

func DoGetVideos(url string, listVideoRes *ListVideoRes) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	body, _ := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(body, listVideoRes); err != nil {
		fmt.Println(err.Error())
	}
}

type Config struct {
	host     string
	pageSize int
	h        bool
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: list_video.exe -host 127.0.0.1 -size 100
`)
	flag.PrintDefaults()
}

var cfg Config

func init() {

	flag.StringVar(&cfg.host, "host", "", "must set host")
	flag.IntVar(&cfg.pageSize, "size", 100, "set pageSize")
	flag.BoolVar(&cfg.h, "h", false, "this help")
	flag.Usage = usage
}

func main() {
	flag.Parse()
	if cfg.h {
		flag.Usage()
		return
	}

	if cfg.host == "" {
		log.Println("please set the host")
		flag.Usage()
		return
	}

	allVideo := make([]Video, 0)
	offset := 0
	videoUrl := "http://%s:8080/v5/videos?pageSize=%d&pageOffset=%d"
	videoCountUrl := "http://%s/api/galaxy/v1/device/cameras:search"
	total := GetVideoCount(fmt.Sprintf(videoCountUrl, cfg.host))
	listVideoResChan := make(chan *ListVideoRes, total/cfg.pageSize+1)
	wg := &sync.WaitGroup{}
	for i := 0; i <= total/cfg.pageSize; i++ {
		wg.Add(1)
		offset = cfg.pageSize * i
		go GetVideos(wg, fmt.Sprintf(videoUrl, cfg.host, cfg.pageSize, offset), listVideoResChan)
	}
	log.Println("等待获取所有设备结果")
	wg.Wait()
	close(listVideoResChan)
	for listVideoRes := range listVideoResChan {
		allVideo = append(allVideo, listVideoRes.Videos...)
	}
	log.Printf("获取到所有的解析设备，解析设备列表大小为%d\n", len(allVideo))
	countMap := make(map[string]int)
	detailMap := make(map[string][]string)
	for _, video := range allVideo {
		countMap[video.Status] += 1
		if video.Status == "VIDEO_PROCESSING" || video.Status == "VIDEO_ERROR" || video.Status == "VIDEO_PREPARING" {
			detailMap[video.Status] = append(detailMap[video.Status], video.Name)
		}
	}
	indent, _ := json.MarshalIndent(countMap, "", "\t")
	log.Println(string(indent))

	indent, _ = json.MarshalIndent(detailMap, "", "\t")
	log.Println(string(indent))
}
