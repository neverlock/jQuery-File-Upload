package main

import (
	//"bytes"
	"fmt"
	//"io/ioutil"
	"log"
	"net/http"
)

func main() {

	key := "Japan"
	//value := "Tokyo"
	//http://localhost:1978/rpc/set?key=japan&value=tokyo
	URL := fmt.Sprintf("http://172.17.0.4:1978/%s", key)

	client := &http.Client{}
	req, err := http.NewRequest("DELETE", URL, nil) // <-- URL-encoded payload
	if err != nil {
		defer func() {
			recover()
			log.Printf("Can'thttp.NewRequest\n")
		}()
		panic(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		defer func() {
			recover()
			log.Printf("Can't client.Do(req)\n")
		}()
		panic(err)
	}
	if !(resp.StatusCode == 200 || resp.StatusCode == 202 || resp.StatusCode == 204) {
		fmt.Println("error with delete")
	}
	//fmt.Println(resp.Status)
	defer resp.Body.Close()

}
