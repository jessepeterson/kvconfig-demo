package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/jessepeterson/kvconfig"
	"io"
	"log"
	"net/http"
	"os"
)

type MDMConfig struct {
	Topic        string `config:"config_topic"`
	AccessRights int    `config:"config_access_rights"`
	BaseURL      string `config:"config_base_url"`
	// MDMURL       string
	// CheckinURL   string
}

type PushCert struct {
	*x509.Certificate `config:"push_cert"`
	*rsa.PrivateKey   `config:"push_pk"`
}

type RuntimeConfiguration struct {
	MDMConfigs    []*MDMConfig
	PushCerts     []*PushCert
	PushCertTest1 *PushCert

	ConfigHTTPPort   int    `config:"cfg_port"`
	ConfigHTTPListen string `config:"cfg_listen"`
	ConfigHTTPUser   string `config:"cfg_user"`
	ConfigHTTPPass   string `config:"cfg_pass"`

	TestValue int `config:"testvalue"`
}

func main() {
	cfgMap := kvconfig.NewMap()

	// first, load our KV store from disk
	if err := cfgMap.ReadEnvFile("test.env"); err != nil {
		log.Fatal(err)
	}

	// next, take any configuration from the environment
	kvconfig.ParseEnv(cfgMap)

	// finally use any CLI arguments
	if err := kvconfig.ParseArgs(cfgMap); err != nil {
		log.Fatal(err)
	}

	// at this point our KV store is populated, lets now generate a configuration
	// structure from the KV store
	runCfg := RuntimeConfiguration{}
	kvconfig.Import(cfgMap, &runCfg)

	// if the listen port is 0 (unconfigured) then set some sane defaults
	if runCfg.ConfigHTTPPort == 0 {
		runCfg.ConfigHTTPPort = 8081
		runCfg.ConfigHTTPListen = "127.0.0.1"
	}

	runCfg.ConfigHTTPUser = "config"

	if runCfg.ConfigHTTPPass == "" || len(runCfg.ConfigHTTPPass) < 10 {
		b := make([]byte, 20)
		rand.Read(b)
		s := sha1.New()
		io.WriteString(s, string(b))
		runCfg.ConfigHTTPPass = hex.EncodeToString(s.Sum(nil))
	}

	listenAndPort := fmt.Sprintf("%s:%d", runCfg.ConfigHTTPListen, runCfg.ConfigHTTPPort)

	// lets export our struct back to the KV store
	kvconfig.Export(&runCfg, cfgMap)

	// and tell our KV store to save to disk
	cfgMap.WriteEnvFile("test.env")

	http.HandleFunc("/api/v1/config", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("config req")
		u, p, ok := r.BasicAuth()

		// check our creds
		if !ok || u != runCfg.ConfigHTTPUser || p != runCfg.ConfigHTTPPass {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.Body == nil {
			http.Error(w, "No request body", 400)
			return
		}

		var resp map[string]int
		err := json.NewDecoder(r.Body).Decode(&resp)

		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		if v, ok := resp["testvalue"]; ok {
			runCfg.TestValue = v

			// lets export our struct back to the KV store
			kvconfig.Export(&runCfg, cfgMap)

			// and tell our KV store to save to disk
			cfgMap.WriteEnvFile("test.env")

			fmt.Println("set test value")
		} else {
			fmt.Println("no testvalue", resp)
		}

		fmt.Fprintf(w, "Hello, %d, %q", 1, r.URL.Path)
	})

	// write file for cfgtool to read and use to contact management interface
	f, err := os.OpenFile("web.cfg", os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Panic(err)
	}
	f.WriteString(fmt.Sprintf("%s:%d:%s:%s\n",
		runCfg.ConfigHTTPListen, runCfg.ConfigHTTPPort,
		runCfg.ConfigHTTPUser, runCfg.ConfigHTTPPass))
	f.Close()
	fmt.Println("configuration contact info written to web.cfg")

	fmt.Println("configuration listening on", listenAndPort)
	http.ListenAndServe(listenAndPort, nil)

}
