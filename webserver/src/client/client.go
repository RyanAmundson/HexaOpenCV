package main

import (
	"log"
	"net/http"
)

func main() {

	resp, err := http.Get("http://127.0.0.1:8000/people")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(resp)
}
