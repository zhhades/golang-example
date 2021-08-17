package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func main() {
	http.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		fmt.Println("path:", r.URL.Path)
		a, _ := strconv.Atoi(r.Form["a"][0])
		b, _ := strconv.Atoi(r.Form["b"][0])
		w.Header().Set("Content-Type", "application/json")
		addData, _ := json.Marshal(map[string]int{
			"data": a + b,
		})
		w.Write(addData)
	})
	http.ListenAndServe(":8000", nil)
}
