package main

import (
	"bytes"
	"fmt"
	//"io/ioutil"
	"net/http"
)

func main() {

	key := "Japan"
	value := "Tokyo"
	//http://localhost:1978/rpc/set?key=japan&value=tokyo
	//URL := fmt.Sprintf("http://172.17.0.4:1978/rpc/set?key=%s&value=%s", key, value)
	Key := fmt.Sprintf("http://172.17.0.4:1978/%s", key)
	client := &http.Client{}
	req, _ := http.NewRequest("PUT", Key, bytes.NewBuffer([]byte(value)))

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println(resp.Status)
	/*
		response, err := http.Get(URL)
		if err != nil {
			fmt.Printf("%s", err)
			//		os.Exit(1)
		} else {
			defer response.Body.Close()
		}
	*/

}
