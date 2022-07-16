package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

var fla = 0

func main() {
	router := httprouter.New()

	router.GET("/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if fla == 20 {
			time.Sleep(4 * time.Second)
			fla = 0
		}
		time.Sleep(1 * time.Second)
		fla++
	})

	fmt.Println("Running...")
	log.Fatal(http.ListenAndServe(":10010", router))
}

type stop struct {
	error
}
