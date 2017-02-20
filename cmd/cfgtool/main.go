package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func parseWebCfg(pathname string) (listen string, port int, username, password string, ok bool) {
	f, err := os.Open(pathname)
	if err != nil {
		ok = false
		return
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)

	split := strings.Split(strings.TrimSpace(string(bytes)), ":")

	if len(split) < 4 {
		ok = false
		return
	}

	port, err = strconv.Atoi(split[1])
	if err != nil {
		ok = false
		return
	}
	listen = split[0]
	username = split[2]
	password = split[3]
	ok = true
	return
}

func main() {
	cfgpath := flag.String("cfgpath", "web.cfg", "path to web config hint file")
	cfgserver := flag.String("cfgserver", "127.0.0.1", "server URL")
	cfgport := flag.Int("cfgport", 8081, "server port")
	cfguser := flag.String("cfguser", "config", "server user")
	cfgpass := flag.String("cfgpass", "", "server password")

	// quite useless example, but this is the test value
	// to have persisted on the other side
	testvalue := flag.Int("testvalue", 0, "server stuff")

	flag.Parse()

	l, port, u, p, ok := parseWebCfg(*cfgpath)

	if ok {
		*cfgserver = l
		*cfgport = port
		*cfguser = u
		*cfgpass = p
	}

	url := fmt.Sprintf("http://%s:%d/api/v1/config", *cfgserver, *cfgport)

	mymap := make(map[string]int)
	fmt.Println("testvalue (change me!) =", *testvalue)
	mymap["testvalue"] = *testvalue
	jsonStr, err := json.Marshal(&mymap)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.SetBasicAuth(*cfguser, *cfgpass)
	client := &http.Client{}
	fmt.Println("submitting to", url)
	resp, err := client.Do(req)
	if err != nil {
		log.Panic(err)
	}
	defer resp.Body.Close()

	fmt.Println(resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("body", string(body))
}
