package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"
)

var (
	clustername = flag.String("clustername", "c1", "download clustername")
)

func ReadLines(fpath string) []string {
	fd, err := os.Open(fpath)
	if err != nil {
		panic(err)
	}
	defer fd.Close()

	var lines []string
	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return lines
}

func Download(clustername string, node string, fileID string) string {
	nt := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s]To download %s\n", nt, fileID)

	url := fmt.Sprintf("http://%s/file/%s", node, fileID)
	fpath := fmt.Sprintf("/yourpath/download/%s_%s", clustername, fileID)
	newFile, err := os.Create(fpath)
	if err != nil {
		fmt.Println(err.Error())
		return "process failed for " + fileID
	}
	defer newFile.Close()

	client := http.Client{Timeout: 900 * time.Second}
	resp, err := client.Get(url)
	defer resp.Body.Close()

	_, err = io.Copy(newFile, resp.Body)
	if err != nil {
		fmt.Println(err.Error())
	}
	return fileID
}

func main() {
	flag.Parse()

	// 从文件中读取节点ip列表
	nodelist := ReadLines(fmt.Sprintf("%s_node.txt", *clustername))
	if len(nodelist) == 0 {
		return
	}

	// 从文件中读取待下载的文件ID列表
	fileIDlist := ReadLines(fmt.Sprintf("%s_fileID.txt", *clustername))
	if len(fileIDlist) == 0 {
		return
	}

	ch := make(chan string)

	// 每个goroutine处理一个文件的下载
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, fileID := range fileIDlist {
		node := nodelist[r.Intn(len(nodelist))]
		go func(node, fileID string) {
			ch <- Download(*clustername, node, fileID)
		}(node, fileID)
	}

	// 等待每个文件下载的完成，并检查超时
	timeout := time.After(900 * time.Second)
	for idx := 0; idx < len(fileIDlist); idx++ {
		select {
		case res := <-ch:
			nt := time.Now().Format("2006-01-02 15:04:05")
			fmt.Printf("[%s]Finish download %s\n", nt, res)
		case <-timeout:
			fmt.Println("Timeout...")
			break
		}
	}
}
