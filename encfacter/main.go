package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type config struct {
	URL string `json:"url"`
}

type facts struct {
	Hostgroup    string `json:"hostgroup"`
	Environment  string `json:"environment"`
	IsProduction bool   `json:"is_production"`
}

var configuration config

func loadConfig(configFile string) (config, error) {
	var config config
	content, _ := ioutil.ReadFile(configFile)
	json.Unmarshal([]byte(content), &config)
	return config, nil
}

func getEncFacts(url string) (facts facts, err error) {
	httpclient := http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Accept", "json")
	res, err := httpclient.Do(req)
	if err != nil {
		return
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &facts)
	if err != nil {
		return
	}
	enhanceFacts(&facts)

	return
}

func enhanceFacts(facts *facts) {
	if facts.Environment == "prodution" && strings.HasPrefix(facts.Hostgroup, "base/Produktion") {
		facts.IsProduction = true
	} else {
		facts.IsProduction = false
	}
}

func writeCache(cacheFile string, facts facts) (err error) {
	content, err := json.MarshalIndent(facts, "", "	")

	err = ioutil.WriteFile(cacheFile, content, 0644)
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(cacheFile), 0755)
		if err != nil {
			return
		}
		writeCache(cacheFile, facts)
	}

	return
}

func readCache(cacheFile string) (facts facts, err error) {
	content, err := ioutil.ReadFile(cacheFile)
	json.Unmarshal(content, &facts)

	return
}

func printFacts(facts facts) {
	json, err := json.MarshalIndent(facts, "", "	")
	if err != nil {
		log.Fatal("Could not print facts")
	}

	fmt.Println(string(json))
}

func main() {
	cacheFile := "/var/cache/encfacter/facts.txt"
	confFile := "/etc/encfacter/config.json"
	var err error

	configuration, err = loadConfig(confFile)
	if err != nil {
		log.Fatalf("Could not read config %s: %v", confFile, err)
		return
	}

	facts, err := getEncFacts(configuration.URL)
	if err != nil {
		facts, err = readCache(cacheFile)
		if err != nil {
			log.Fatalf("Could not read cache %s: %v", cacheFile, err)
		}
	} else {
		err = writeCache(cacheFile, facts)
		if err != nil {
			log.Fatalf("Could not write cache %s: %v", cacheFile, err)
		}
	}

	printFacts(facts)
}
