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
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

func DownloadFile(URL string, fileName string) {
	resp, _ := http.Get(URL)
	body, _ := ioutil.ReadAll(resp.Body)
	out, _ := os.Create(fileName)
	defer out.Close()
	_, err := io.Copy(out, bytes.NewReader(body))
	if err != nil {
		log.Printf("下载文件报错[%s],error is %v", URL, err)
	}
}

func DownloadImage(imageURL string, workDirPath string, caseNumber string) {
	imageName := workDirPath + SLASH + caseNumber + JPG
	DownloadFile(imageURL, imageName)
	log.Printf("download event [%s] image success...", caseNumber)
}

func DownloadVideo(videoURL string, workDirPath string, caseNumber string) {
	videoName := workDirPath + SLASH + caseNumber + MP4
	DownloadFile(videoURL, videoName)
}

func ZipDir(dir, zipFile string) {

	fz, err := os.Create(zipFile)
	if err != nil {
		log.Fatalf("Create zip file failed: %s\n", err.Error())
	}
	defer fz.Close()

	w := zip.NewWriter(fz)
	defer w.Close()

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			fDest, err := w.Create(path[len(dir)+1:])
			if err != nil {
				log.Printf("Create failed: %s\n", err.Error())
				return nil
			}
			fSrc, err := os.Open(path)
			if err != nil {
				log.Printf("Open failed: %s\n", err.Error())
				return nil
			}
			defer fSrc.Close()
			_, err = io.Copy(fDest, fSrc)
			if err != nil {
				log.Printf("Copy failed: %s\n", err.Error())
				return nil
			}
		}
		return nil
	})
}

func Decimal(value float64) float64 {
	value, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return value
}

func DoWork(doc Document, config CoreConfig, downloadParentPath string,
	alreadyDownloadCaseNumber []string, wg *sync.WaitGroup, currentCount *AtomicInt, total int) {
	defer wg.Done()
	event := doc.Source
	caseNumber := event["caseNumber"].(string)
	if IsContain(alreadyDownloadCaseNumber, caseNumber) {
		log.Printf("caseNumber [%s] already download", caseNumber)
		currentCount.Increment()
		log.Printf("下载进度.............................................%d%%", 100*currentCount.get()/total)
		return
	} else {
		workDirPath := downloadParentPath + SLASH + caseNumber
		if _, err := os.Stat(workDirPath); os.IsNotExist(err) {
			os.Mkdir(workDirPath, os.ModePerm)
		}
		//1、下载图片
		causeImageUri := event["causeImage"].(map[string]interface{})["fileUri"].(string)
		causeVideoUri := event["causeVideo"].(map[string]interface{})["fileUri"].(string)
		DownloadImage(fmt.Sprintf(config.ImageURLPrefix, causeImageUri), workDirPath, caseNumber)
		//2、下载视频
		if config.VideoDownloadFlag {
			DownloadVideo(fmt.Sprintf(config.VideoURLPrefix, causeVideoUri), workDirPath, caseNumber)
			log.Printf("download event [%s] video success...", caseNumber)
		}

		// 压缩目录
		if config.ZipFlag {
			ZipDir(workDirPath, downloadParentPath+SLASH+caseNumber+".zip")
			if err := os.RemoveAll(workDirPath); err != nil {
				log.Printf("删除目录失败[%s] err is [%v]", workDirPath, err)
			}
		}

		//3、写入告警信息
		if jsonStr, err := json.Marshal(event); err != nil {
			log.Printf("事件转json出错[%v]", event)
		} else {
			WriteContentToFile(string(jsonStr), downloadParentPath+SLASH+"alarm_info.txt")
		}
		//4、保存下载记录
		WriteContentToFile(caseNumber, downloadParentPath+SLASH+"alarm_download_already.txt")
		log.Printf("event [%s] download success...", caseNumber)
		currentCount.Increment()
		log.Printf("下载进度.............................................%d%%", 100*currentCount.get()/total)
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
			log.Fatalf("初始化文件失败[%s]", fileName)
		}
	}
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("文件打开失败", err)
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
		log.Printf("下载目录已存在[%s]", path)
	} else {
		if err := os.Mkdir(path, os.ModePerm); err != nil {
			log.Fatalf("创建下载目录[%s]失败[%v]", path, err)
		}
		log.Printf("成功创建下载目录[%s]", path)
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
			log.Printf("读取文件失败")
		}
	} else {
		file, err = os.Create(fileName)
		file.Close()
		file, err = os.Open(fileName)
	}
	if err != nil {
		log.Printf("读取文件失败")
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		res = append(res, scanner.Text())
	}
	return res
}

func splitArray(arr []Document, num int64) [][]Document {
	max := int64(len(arr))
	if max < num {
		return nil
	}
	var segments = make([][]Document, 0)
	quantity := max / num
	end := int64(0)
	for i := int64(1); i <= num; i++ {
		qu := i * quantity
		if i != num {
			segments = append(segments, arr[i-1+end:qu])
		} else {
			segments = append(segments, arr[i-1+end:])
		}
		end = qu - i
	}
	return segments
}

func main() {
	start := time.Now()
	v := viper.New()
	v.SetConfigFile("alarm_download.yaml")
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("读取配置文件失败 %v", err.Error())
	}

	config := Config{}
	if err := v.Unmarshal(&config); err != nil {
		log.Fatalf("获取配置映射失败 %v", err.Error())
	}
	if prettyJSON, err := json.MarshalIndent(config, "", "\t"); err != nil {
		log.Fatalf("json序列化失败 %v", err.Error())
	} else {
		log.Printf("读取配置信息为：%v", string(prettyJSON))
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
	wg.Add(totalCount)
	for _, doc := range docs {
		go DoWork(doc, config.Core, downloadParentPath, downloadedEventArr, &wg, &currentCount, totalCount)
	}
	wg.Wait()

	log.Printf("总共耗时[%s]", time.Since(start))
}
