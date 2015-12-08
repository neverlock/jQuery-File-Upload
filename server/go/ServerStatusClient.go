package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/neverlock/utility/selfupdate"
	"github.com/robfig/cron"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	//"strconv"
	//"strings"
	"time"
)

type ServerData struct {
	ProjectName  string
	UUID         string
	Version      string
	Uptime       string
	Network      string
	Df           string
	Disk         string
	LastLogFile  string
	ClientConfig string
}

func main() {
	t := time.Now()
	fmt.Println("Start time =", t.Format("2006/01/02 15:04:05"))
	WriteConfig("system", "starttime", t.Format("2006/01/02 15:04:05")) //memo start time

	LoadConfig()
	go selfupdate.SelfUpdate(cfg.System.Version, cfg.System.UpdateURL, cfg.System.AppName)

	fmt.Println("Update Status to server")
	fmt.Println("Use CTRL+C to Exit")
	c := cron.New()
	//	c.AddFunc(cfg.Copycat.Cron, func() { getCopycatQueue() })
	c.AddFunc(cfg.System.Cron, func() { UpdateStatus() })
	go c.Start()
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)
	<-sig

	//getCopycatQueue()
}

func UpdateStatus() {
	var Data ServerData
	Data.ProjectName = cfg.System.ProjectName
	Data.UUID = cfg.System.UUID
	Data.Version = cfg.System.Version
	Data.Uptime = execmd("uptime", "")
	Data.Network = execmd("ifconfig", "")
	Data.Df = execmd(cfg.Cmd1.Cmd, cfg.Cmd1.Arg)
	Data.Disk = execmd(cfg.Cmd2.Cmd, cfg.Cmd2.Arg)
	Data.LastLogFile = execmd(cfg.Cmd3.Cmd, cfg.Cmd3.Arg)
	Data.ClientConfig = execmd(cfg.Cmd4.Cmd, cfg.Cmd4.Arg)
	js, _ := json.Marshal(Data)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", cfg.System.StatusServer, bytes.NewBuffer(js))
	req.Header.Set("X-Custom-Header", "ServerStatus")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println(resp.Status)

	//make struct for keep value
	//assign value from cfg struct to new struct
	//get uptime
	//run cmd1-4
	//post to update server
}

func execmd(cmd string, arg string) string {
	CMD := fmt.Sprintf("%s %s", cmd, arg)
	exe := exec.Command("/bin/sh", "-c", CMD)
	var out bytes.Buffer
	var stderr bytes.Buffer
	exe.Stdout = &out
	exe.Stderr = &stderr
	err := exe.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		log.Fatal(err)
	}
	return out.String()
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}
