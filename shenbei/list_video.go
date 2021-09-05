package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

type Video struct {
	Url    string `json:"url"`
	Status string `json:"status"`
	ID     string `json:"id"`
	Name   string `json:"name"`
}

type ListVideoRes struct {
	Videos        []Video `json:"videos"`
	NextPageToken string  `json:"nextPageToken"`
}

func GetVideos(url string, listVideoRes *ListVideoRes) {
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

func main() {
	allVideo := make([]Video, 0)
	pageSize := 1000
	offset := 0
	url := "http://172.99.3.229:8080/v5/videos?pageSize=%d&pageOffset=%d"
	listVideoRes := &ListVideoRes{}
	GetVideos(fmt.Sprintf(url, pageSize, offset), listVideoRes)
	allVideo = append(allVideo, listVideoRes.Videos...)
	for len(listVideoRes.Videos) > 0 {
		offset += pageSize
		tmpUrl := fmt.Sprintf(url, pageSize, offset)
		GetVideos(tmpUrl, listVideoRes)
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
