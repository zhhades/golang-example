package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/spf13/viper"
)

const SLASH = "/"
const JPG = ".jpg"
const MP4 = ".mp4"

type ESConfig struct {
	Host        string `json:"host" mapstructure:"host"`
	User        string `json:"user" mapstructure:"user"`
	Password    string `json:"password" mapstructure:"password"`
	SearchIndex string `json:"searchIndex" mapstructure:"search_index"`
	SearchDsl   string `json:"searchDsl" mapstructure:"search_dsl"`
}

type CoreConfig struct {
	ImageURLPrefix    string `json:"imageURLPrefix" mapstructure:"image_url_prefix"`
	VideoURLPrefix    string `json:"videoURLPrefix" mapstructure:"video_url_prefix"`
	VideoDownloadFlag bool   `json:"VideoDownloadFlag" mapstructure:"video_download_flag"`
	ZipFlag           bool   `json:"zipFlag" mapstructure:"zip_flag"`
}

type Config struct {
	ES   ESConfig   `json:"es" mapstructure:"es"`
	Core CoreConfig `json:"core" mapstructure:"core"`
}

type HitsResult struct {
	Hits     []Document             `json:"hits"`
	MaxScore float64                `json:"max_score"`
	Total    map[string]interface{} `json:"total"`
}

type Document struct {
	Id     string                 `json:"_id"`
	Index  string                 `json:"_index"`
	Score  float64                `json:"_score"`
	Source map[string]interface{} `json:"_source"`
}

type EsSearchRes struct {
	Took    uint64                 `json:"took"`
	Shard   map[string]interface{} `json:"_shards"`
	TimeOut bool                   `json:"time_out"`
	Hits    HitsResult             `json:"hits"`
}

type AtomicInt struct {
	value int
	lock  sync.Mutex
}

func DownloadFile(url, filename string) {
	r, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()

	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, _ = io.Copy(f, r.Body)

}

func (a *AtomicInt) Increment() {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.value++
}

func (a *AtomicInt) get() int {
	a.lock.Lock()
	defer a.lock.Unlock()
	return a.value
}

func InitESClient(config ESConfig) *elasticsearch.Client {
	conf := elasticsearch.Config{
		Addresses: []string{config.Host},
		Username:  config.User,
		Password:  config.Password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	esClient, err := elasticsearch.NewClient(conf)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}
	return esClient
}

func JsonToMap(jsonStr string) map[string]interface{} {
	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonStr), &m)
	if err != nil {
		log.Printf("Unmarshal with error: %+v\n", err)
		return nil
	}
	return m
}

func QueryESDSL(client *elasticsearch.Client, index string, dsl interface{}) *esapi.Response {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(dsl); err != nil {
		log.Fatal(err, "Error encoding query")
	}
	res, err := client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex(index),
		client.Search.WithBody(&buf),
		client.Search.WithTrackTotalHits(true),
		client.Search.WithPretty(),
	)
	if err != nil {
		log.Fatal(err, "Error getting response")
	}

	log.Printf("search es response status %d", res.StatusCode)

	return res
}

func IsContain(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

func DownloadImage(imageURL string, workDirPath string, caseNumber string) {
	imageName := workDirPath + SLASH + caseNumber + JPG
	DownloadFile(imageURL, imageName)
	log.Printf("download event [%s] image success...", caseNumber)
}

func DownloadVideo(videoURL string, workDirPath string, caseNumber string) {
	videoName := workDirPath + SLASH + caseNumber + MP4
	DownloadFile(videoURL, videoName)
	log.Printf("<<<<<download event [%s] video success>>>>>", caseNumber)
}

func Zip(dst, src string) (err error) {
	// ???????????????????????????
	fw, err := os.Create(dst)
	defer fw.Close()
	if err != nil {
		return err
	}

	// ?????? fw ????????? zip.Write
	zw := zip.NewWriter(fw)
	defer func() {
		// ??????????????????????????????
		if err := zw.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

	// ???????????????????????? zw ?????????????????????????????????????????????????????????????????????
	return filepath.Walk(src, func(path string, fi os.FileInfo, errBack error) (err error) {
		if errBack != nil {
			return errBack
		}

		// ??????????????????????????? zip ???????????????
		fh, err := zip.FileInfoHeader(fi)
		if err != nil {
			return
		}

		// ?????????????????????????????????
		fh.Name = strings.TrimPrefix(path, string(filepath.Separator))

		// ?????????????????????????????????????????????????????????????????????
		if fi.IsDir() {
			fh.Name += "/"
		}

		// ???????????????????????????????????? Write ??????
		w, err := zw.CreateHeader(fh)
		if err != nil {
			return
		}

		// ????????????????????????????????????????????????????????????????????????????????? w
		// ????????????????????????????????????
		if !fh.Mode().IsRegular() {
			return nil
		}

		// ????????????????????????
		fr, err := os.Open(path)
		defer fr.Close()
		if err != nil {
			return
		}

		// ?????????????????? Copy ??? w
		n, err := io.Copy(w, fr)
		if err != nil {
			return
		}
		// ?????????????????????
		fmt.Printf("????????????????????? %s, ???????????? %d ??????????????????\n", path, n)

		return nil
	})
}
func DoWork(wg *sync.WaitGroup, doc Document, config CoreConfig, downloadParentPath string,
	alreadyDownloadCaseNumber []string, currentCount *AtomicInt, total int) {
	defer wg.Done()
	event := doc.Source
	caseNumber := event["caseNumber"].(string)
	if IsContain(alreadyDownloadCaseNumber, caseNumber) {
		log.Printf("caseNumber [%s] already download", caseNumber)
		currentCount.Increment()
		log.Printf("????????????.............................................%d%%", 100*currentCount.get()/total)
		return
	} else {
		workDirPath := downloadParentPath + SLASH + caseNumber
		if _, err := os.Stat(workDirPath); os.IsNotExist(err) {
			os.Mkdir(workDirPath, os.ModePerm)
		}
		//1???????????????
		causeImageUri := event["causeImage"].(map[string]interface{})["fileUri"].(string)
		causeVideoUri := event["causeVideo"].(map[string]interface{})["fileUri"].(string)
		DownloadImage(fmt.Sprintf(config.ImageURLPrefix, causeImageUri), workDirPath, caseNumber)
		//2???????????????
		if config.VideoDownloadFlag {
			DownloadVideo(fmt.Sprintf(config.VideoURLPrefix, causeVideoUri), workDirPath, caseNumber)
			log.Printf("download event [%s] video success...", caseNumber)
		}

		//3?????????????????????
		if jsonStr, err := json.Marshal(event); err != nil {
			log.Printf("?????????json??????[%v]", event)
		} else {
			WriteContentToFile(string(jsonStr), downloadParentPath+SLASH+"alarm_info.txt")
		}
		//4?????????????????????
		WriteContentToFile(caseNumber, downloadParentPath+SLASH+"alarm_download_already.txt")
		log.Printf("event [%s] download success...", caseNumber)
		currentCount.Increment()
		log.Printf("????????????.............................................%d%%", 100*currentCount.get()/total)
		return
	}

}

func WriteContentToFile(content string, fileName string) {
	m := sync.Mutex{}
	m.Lock()
	defer m.Unlock()
	if !Exists(fileName) {
		file, err := os.Create(fileName)
		defer file.Close()
		if err != nil {
			log.Fatalf("?????????????????????[%s]", fileName)
		}
	}
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("??????????????????", err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	writer.WriteString(content + "\n")
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func InitDir(path string) {
	if Exists(path) {
		log.Printf("?????????????????????[%s]", path)
	} else {
		if err := os.Mkdir(path, os.ModePerm); err != nil {
			log.Fatalf("??????????????????[%s]??????[%v]", path, err)
		}
		log.Printf("????????????????????????[%s]", path)
	}
}

func ReadFileTransferToEventCaseNumberArr(fileName string) []string {
	var (
		res  []string
		file *os.File
		err  error
	)
	if Exists(fileName) {
		file, err = os.Open(fileName)
		if err != nil {
			log.Printf("??????????????????")
		}
	} else {
		file, err = os.Create(fileName)
		file.Close()
		file, err = os.Open(fileName)
	}
	if err != nil {
		log.Printf("??????????????????")
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		res = append(res, scanner.Text())
	}
	return res
}

func main() {
	start := time.Now()
	v := viper.New()
	v.SetConfigFile("alarm_download.yaml")
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("???????????????????????? %v", err.Error())
	}

	config := Config{}
	if err := v.Unmarshal(&config); err != nil {
		log.Fatalf("???????????????????????? %v", err.Error())
	}
	if prettyJSON, err := json.MarshalIndent(config, "", "\t"); err != nil {
		log.Fatalf("json??????????????? %v", err.Error())
	} else {
		log.Printf("????????????????????????%v", string(prettyJSON))
	}

	downloadParentPath := "download" + time.Now().Format("20060102")
	InitDir(downloadParentPath)

	esClient := InitESClient(config.ES)
	res := QueryESDSL(esClient, config.ES.SearchIndex, JsonToMap(config.ES.SearchDsl))
	defer res.Body.Close()

	var esSearchRes EsSearchRes
	if err := json.NewDecoder(res.Body).Decode(&esSearchRes); err != nil {
		log.Printf("Error parsing the response body: %s", err)
	}

	downloadedEventArr := ReadFileTransferToEventCaseNumberArr(downloadParentPath + SLASH + "alarm_download_already.txt")
	docs := esSearchRes.Hits.Hits

	currentCount := AtomicInt{
		value: 0,
		lock:  sync.Mutex{},
	}
	totalCount := len(docs)
	wg := sync.WaitGroup{}
	wg.Add(len(docs))
	for _, doc := range docs {
		go DoWork(&wg, doc, config.Core, downloadParentPath, downloadedEventArr, &currentCount, totalCount)
	}
	wg.Wait()

	// ????????????
	if config.Core.ZipFlag {
		log.Printf("??????????????????[%s]>>>>>>", downloadParentPath)
		Zip(downloadParentPath+".zip", downloadParentPath)
		log.Printf("??????????????????[%s]>>>>>>", downloadParentPath+".zip")
		if err := os.RemoveAll(downloadParentPath); err != nil {
			log.Printf("??????????????????[%s] err is [%v]", downloadParentPath, err)
		}
	}

	log.Printf("????????????[%s]", time.Since(start))
}
