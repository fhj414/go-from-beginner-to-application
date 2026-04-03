// Package main: 入门示例 — 约 10 分钟可跑起来的最小 HTTP 服务。
package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "Hello, Go!")
	})

	addr := ":8080"
	log.Printf("listening on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
