package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"os"
	"time"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

type Config struct {
	DefaultScrapeTime	int	   `yaml:"DefaultScrapeTime"`
	Hosts []struct {
		HostName string `yaml:"HostName"`
		ScrapeTime int `yaml:"ScrapeTime"`
		HostPaths []string `yaml:"HostPaths"`
		Plugins []string `yaml:"Plugins"`
	} `yaml:"Hosts"`

	DefaultPlugins	[]string   `yaml:"DefaultPlugins"`
}


type Check struct {
        ConfigLabel string
	Host string
        TimeStamp string
        EpochTime int64
        Command string
        Output string
        Retval int
}

var configs []Config
var checks []Check

func MakeSkel() error {
	err := os.MkdirAll("/etc/heimdall/scraper.d/", 0644)
	if err != nil {
		return err
	}

	file, err := os.OpenFile("/etc/heimdall/scraper.d/default.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString("# Scrape Time If Not Defined In A Specific Host\n")
	file.WriteString("DefaultScrapeTime: 600\n\n")
	file.WriteString("# List Of Hosts To Check.\n")
	file.WriteString("Hosts:\n")
	file.WriteString("  # The Hostname/IP To Check\n")
	file.WriteString("  - HostName: localhost\n\n")
	file.WriteString("    #   The Wait Timeout To Scrape This Specific Host\n")
	file.WriteString("    ScrapeTime: 600\n\n")
	file.WriteString("    #   The URL Path On The Server To Check (defaults to /checkandclear)\n")
	file.WriteString("    #   This Can Be More Than One\n")
	file.WriteString("    HostPaths:\n")
	file.WriteString("      - /checkandclear\n\n")
	file.WriteString("    #   The Plugins To Run After Scraping This Host\n")
	file.WriteString("    Plugins:\n")
	file.WriteString("      - splunk\n\n")
	file.WriteString("# The Plugins To Run If Not Specified In The Host Block\n")
	file.WriteString("DefaultPlugins:\n")
	file.WriteString("  - splunk\n")
	file.WriteString("  - rules\n")
	file.WriteString("  - influxdb\n")

	file.Close()

	return nil
}

func Log(message string) {
	file, err := os.OpenFile("./heimdall_scraper.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed To Open Log File: " + err.Error())
	}
	file.Close()

	current_time := time.Now().Local()
	t := current_time.Format("Jan 02 2006 03:04:05")
	_, err = file.WriteString(t + " - Heimdall Scraper: " + message + "\n")
}

func GetConfigs() {

	_, err := os.Stat("/etc/heimdall/scraper.d/")
	if os.IsNotExist(err) {
		err := MakeSkel()
		if err != nil {
			fmt.Println("Error Setting Up /etc/heimdall/scraper.d/ and default settings file: " + err.Error())
			Log("Error Setting Up /etc/heimdall/scraper.d/ and default settings file: " + err.Error())
			return
		}
	}

	files, err := ioutil.ReadDir("/etc/heimdall/scraper.d/")
	if err != nil {
		fmt.Println("Error Reading /etc/heimdall/scraper.d/: " + err.Error())
		Log("Error Reading /etc/heimdall/scraper.d/: " + err.Error())
		return
	}

	if len(files) < 1 {
		fmt.Println("/etc/heimdall/scraper.d/ exists, but is empty. No Configs Loaded")
		Log("/etc/heimdall/scraper.d/ exists, but is empty. No Configs Loaded")
	}

	for _, f := range files {
		b, err := ioutil.ReadFile("/etc/heimdall/scraper.d/" + f.Name())
		if err != nil {
			fmt.Println("Error Opening File: /etc/heimdall/scraper.d/" + f.Name() + ": " + err.Error())
			Log("Error Opening File: /etc/heimdall/scraper.d/" + f.Name() + ": " + err.Error())
		}

		yml := string(b)

		c := Config{}
		err = yaml.Unmarshal([]byte(yml), &c)

		if err != nil {
			fmt.Println("Couldn't Parse YAML File /etc/heimdall/scraper.d/" + f.Name() + ": " + err.Error())
			Log("Couldn't Parse YAML File /etc/heimdall/scraper.d/" + f.Name() + ": " + err.Error())
		}

		configs = append(configs, c)
	}

}

func handleWhoAreYou(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Heimdall Scraper")
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

func handleChecks(w http.ResponseWriter, r *http.Request) {
	jsn, _ := json.Marshal(checks)
	fmt.Fprintf(w, string(jsn))
}

func handleCheckAndClear(w http.ResponseWriter, r *http.Request) {
	jsn, _ := json.Marshal(checks)
	fmt.Fprintf(w, string(jsn))
	checks = nil
}

func handleStatusOf(w http.ResponseWriter, r *http.Request) {
	retvals := []Check{}
	check := r.URL.Query().Get("service")
	if len(check) == 0 {
		fmt.Fprintf(w, "missing service to retrieve")
		return
	}

	for _, chk := range checks {
		if chk.ConfigLabel == check {
			retvals = append(retvals, chk)
		}
	}

	jsn, _ := json.Marshal(retvals)
	fmt.Fprintf(w, string(jsn))
}

func Do_Scrapes(c *Config, chanl chan Check) {
	var check Check

	if c.DefaultScrapeTime < 1 {
		c.DefaultScrapeTime = 300
	}

	for {
		time.Sleep(time.Duration(c.DefaultScrapeTime) * time.Second)
		for i := 0; i < len(c.Hosts); i++ {
			//path := ""
		}
		chanl<-check
	}
}

func main() {
	GetConfigs()

	go func() {

		chanl := make(chan Check)
		for i := 0; i < len(configs); i++ {
			c := configs[i]
			//if c.Enabled {
				go Do_Scrapes(&c, chanl)
		//	}
		}

		for {
			tmp := <-chanl
			checks = append(checks, tmp)
		}
		
	}()

	fmt.Println(configs)

	router := mux.NewRouter()
	router.HandleFunc("/whoareyou", handleWhoAreYou)
	router.HandleFunc("/ping", handlePing)
	router.HandleFunc("/checks", handleChecks)
	router.HandleFunc("/checkandclear", handleCheckAndClear)
	router.HandleFunc("/statusof", handleStatusOf)

	err := http.ListenAndServe(":80", router)
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
	}
}
