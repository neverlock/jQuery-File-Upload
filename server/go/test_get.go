package main

import (
	//"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

func main() {

	key := "Japan"
	//http://localhost:1978/rpc/set?key=japan&value=tokyo
	URL := fmt.Sprintf("http://172.17.0.4:1978/%s", key)
	response, err := http.Get(URL)
	if err != nil {
		fmt.Printf("%s", err)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
		}
		fmt.Printf("%s\n", string(contents))
	}

}
