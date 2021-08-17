package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/olivere/elastic/v7"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const mappingTpl = `{
    "mappings":{
        "properties":{
            "id":                 { "type": "long" },
            "username":         { "type": "keyword" },
            "nickname":            { "type": "text" },
            "phone":            { "type": "keyword" },
            "age":                { "type": "long" },
            "update_time":        { "type": "long" },
            "create_time":        { "type": "long" }
            }
        }
    }`
const author = "zhhades"
const project = "es-example"

type UserES struct {
	client  *elastic.Client
	index   string
	mapping string
}

type User struct {
	ID         uint64 `json:"id"`
	Username   string `json:"username"`
	Nickname   string `json:"nickname"`
	Phone      string `json:"phone"`
	Age        uint8  `json:"age"`
	UpdateTime uint64 `json:"update_time"`
	CreateTime uint64 `json:"create_time"`
}

func (es *UserES) BatchAdd(ctx context.Context, user []*User) error {
	var err error
	if err = es.batchAdd(ctx, user); err != nil {
		fmt.Println("batch add failed ", err)
	}
	return err
}

func (es *UserES) batchAdd(ctx context.Context, user []*User) error {
	req := es.client.Bulk().Index(es.index)
	for _, u := range user {
		u.UpdateTime = uint64(time.Now().UnixNano()) / uint64(time.Millisecond)
		u.CreateTime = uint64(time.Now().UnixNano()) / uint64(time.Millisecond)
		doc := elastic.NewBulkIndexRequest().Id(strconv.FormatUint(u.ID, 10)).Doc(u)
		req.Add(doc)
	}
	if req.NumberOfActions() < 0 {
		return nil
	}
	if _, err := req.Do(ctx); err != nil {
		return err
	}
	return nil
}

func NewUserES(client *elastic.Client) *UserES {
	index := fmt.Sprintf("%s_%s", author, project)
	userEs := &UserES{
		client:  client,
		index:   index,
		mapping: mappingTpl,
	}

	userEs.init()

	return userEs
}

func (es *UserES) init() {
	ctx := context.Background()

	exists, err := es.client.IndexExists(es.index).Do(ctx)
	if err != nil {
		fmt.Printf("userEs init exist failed err is %s\n", err)
		return
	}

	if !exists {
		_, err := es.client.CreateIndex(es.index).Body(es.mapping).Do(ctx)
		if err != nil {
			fmt.Printf("userEs init failed err is %s\n", err)
			return
		}
	}
}

func makeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}

func splitArray(arr []*User, num int64) [][]*User {
	max := int64(len(arr))
	if max < num {
		return nil
	}
	var segmens = make([][]*User, 0)
	quantity := max / num
	end := int64(0)
	for i := int64(1); i <= num; i++ {
		qu := i * quantity
		if i != num {
			segmens = append(segmens, arr[i-1+end:qu])
		} else {
			segmens = append(segmens, arr[i-1+end:])
		}
		end = qu - i
	}
	return segmens
}

func main() {

	var users []*User

	for _, v := range makeRange(1, 1000000) {
		user := &User{
			ID:       uint64(v),
			Username: fmt.Sprintf("ZHHADES-%d", v),
			Age:      uint8(v),
			Nickname: fmt.Sprintf("zhhades-%d", v),
			Phone:    strconv.Itoa(v + 13000000000),
		}
		users = append(users, user)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			TLSHandshakeTimeout: 3000 * time.Second,
		},
		Timeout: time.Second * 30000,
	}
	esClient, err := elastic.NewClient(
		elastic.SetHttpClient(httpClient),
		elastic.SetURL("https://10.122.100.111:9200"),
		elastic.SetSniff(false),
		elastic.SetBasicAuth("elastic", "QcYfRDraTCr"),
		elastic.SetErrorLog(log.New(os.Stderr, "ELASTIC ", log.LstdFlags)),
		elastic.SetInfoLog(log.New(os.Stdout, "ELASTIC ", log.LstdFlags)),
	)
	if err != nil {
		log.Fatal(err)
	}
	userES := NewUserES(esClient)

	now := time.Now()
	for _, v := range splitArray(users, 10) {
		err = userES.BatchAdd(context.Background(), v)
		if err != nil {
			panic(err.Error)
		}
	}
	fmt.Println(time.Since(now))

}
