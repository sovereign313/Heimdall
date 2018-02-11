package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"time"
	"path/filepath"
	"plugin"
	"errors"
)

type Config struct {
	AutoClear bool `yaml:"AutoClear"`
	DefaultScrapeTime int    `yaml:"DefaultScrapeTime"`
	PluginPath        string `yaml:"PluginPath"`
	Hosts             []struct {
		HostName       string   `yaml:"HostName"`
		ScrapeTime     int      `yaml:"ScrapeTime"`
		HostPaths      []string `yaml:"HostPaths"`
		Plugins        []string `yaml:"Plugins"`
		FailurePlugins []string `yaml:"FailurePlugins"`
	} `yaml:"Hosts"`

	DefaultPlugins []string `yaml:"DefaultPlugins"`
}

type Check struct {
	ConfigLabel string `json:"ConfigLabel"`
	Host        string `json:"Host"`
	TimeStamp   string `json:"TimeStamp"`
	EpochTime   int64  `json:"Epochtime"`
	Command     string `json:"Command"`
	Output      string `json:"Output"`
	Retval      int    `json:"Retval"`
}

type PluginList struct {
        Name string
        Version string
        Function func(string, bool) (bool, error)
}

var configs []Config
var checks []Check
var plugins []PluginList


func MakeSkel() error {
	err := os.MkdirAll("/etc/heimdall/scraper.d/", 0644)
	if err != nil {
		return err
	}

	err = os.MkdirAll("/etc/heimdall/plugins.d/", 0644)
	if err != nil {
		return err
	}

	file, err := os.OpenFile("/etc/heimdall/scraper.d/default.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString("# Automatically Clear Checks List When Processed?\n")
	file.WriteString("# Set To \"false\" if this scraper is a sub-scraper node.\n")
	file.WriteString("AutoClear: true\n\n")
	file.WriteString("# Scrape Time If Not Defined In A Specific Host\n")
	file.WriteString("DefaultScrapeTime: 600\n\n")
	file.WriteString("# The Path To The Plugins (.so files)\n")
	file.WriteString("PluginPath: ./plugins\n\n")
	file.WriteString("# List Of Hosts To Check.\n")
	file.WriteString("Hosts:\n")
	file.WriteString("  # The Hostname/IP To Check\n")
	file.WriteString("  - HostName: localhost:9050\n\n")
	file.WriteString("    #   The Wait Timeout To Scrape This Specific Host\n")
	file.WriteString("    ScrapeTime: 600\n\n")
	file.WriteString("    #   The URL Path On The Server To Check (defaults to /checkandclear)\n")
	file.WriteString("    #   This Can Be More Than One\n")
	file.WriteString("    HostPaths:\n")
	file.WriteString("      - /checkandclear\n\n")
	file.WriteString("    #   The Plugins To Run After Scraping This Host\n")
	file.WriteString("    Plugins:\n")
	file.WriteString("      - splunk\n\n")
	file.WriteString("    # What To Run If Scraping Fails\n")
	file.WriteString("    FailurePlugins:\n")
	file.WriteString("      - alert_smtp\n\n")
	file.WriteString("# The Plugins To Run If Not Specified In The Host Block\n")
	file.WriteString("DefaultPlugins:\n")
	file.WriteString("  - splunk\n")
	file.WriteString("  - rules\n")
	file.WriteString("  - influxdb\n")
	file.Close()

	sfile, err := os.OpenFile("/etc/heimdall/plugins.d/splunk.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	sfile.WriteString("# Host:Port Of Splunk Server To Send To\n")
	sfile.WriteString("Host: splunkserver:5021\n")
	sfile.Close()

	rfile, err := os.OpenFile("/etc/heimdall/plugins.d/rules.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	rfile.WriteString("# Host:Port Of Rules Server To Send To\n")
	rfile.WriteString("Host: 127.0.0.1:8225\n")
	rfile.Close()

	ffile, err := os.OpenFile("/etc/heimdall/plugins.d/alert_smtp.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	ffile.WriteString("# Who To Alert In The Event Of A Failed Scrape\n")
	ffile.WriteString("AlertList:\n")
	ffile.WriteString("  - someone@yourdomain.com\n\n")
	ffile.WriteString("# Who The Email Should Come From\n")
	ffile.WriteString("FromAddress: heimdall@yourdomain.com\n\n")
	ffile.WriteString("# The Subject For The Failed Scrape Email\n")
	ffile.WriteString("Subject: Failed To Scrape Host\n\n")
	ffile.WriteString("# The SMTP Server You Want To Use As The MTA\n")
	ffile.WriteString("SMTPServer: smtp.mydomain.com\n")
	ffile.Close()

	return nil
}

func Log(message string) {
	file, err := os.OpenFile("./heimdall_scraper.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed To Open Log File: " + err.Error())
	}
	defer file.Close()

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

func LoadPlugins(plgpath string) error {
	_, er := os.Stat(plgpath)
	if os.IsNotExist(er) {
		Log("Plugin Path Doesn't Exist (" + plgpath + ")")
		Log("No Plugins Loaded")
		fmt.Println("No Plugins Loaded")
		return er
	}

        all_plugins, err := filepath.Glob(plgpath + "/*.so")
        if err != nil {
		Log("Error Getting Files From: " + plgpath + ": " + err.Error())
		return err
        }

        for _, filename := range all_plugins {
                p, err := plugin.Open(filename)
                if err != nil {
			return err
                }

                symbol, err := p.Lookup("Handle")
		if err != nil {
			Log("failed to look up Function: " + err.Error())
			Log("Plugin Not Loaded: " + filename)
			continue
		}

                nsymbol, err := p.Lookup("PluginName")
		if err != nil {
			Log("failed to look up Plugin Name: " + err.Error())
			Log("Plugin Not Loaded: " + filename)
			continue
		}

                vsymbol, err := p.Lookup("PluginVersion")
                if err != nil {
			Log("failed to look up Plugin Version: " + err.Error())
			Log("Plugin Not Loaded: " + filename)
			continue
                }

                plgname, ok := nsymbol.(*string)
		if !ok {
			Log("failed to load name symbol from: " + filename)
			Log("Plugin Not Loaded")
			continue
		}

                plgversion, ok := vsymbol.(*string)
		if !ok {
			Log("failed to load name symbol from: " + filename)
			Log("Plugin Not Loaded")
			continue
		}

                plgfunc, ok := symbol.(func(string, bool) (bool, error))
                if !ok {
			Log("failed to load name symbol from: " + filename)
			Log("Plugin Not Loaded")
			continue
                }

                tmpplg := PluginList{}
                tmpplg.Name = *plgname
                tmpplg.Version = *plgversion
                tmpplg.Function = plgfunc

		flag := false
		for _, p := range plugins {
			if p.Name == tmpplg.Name {
				Log("Plugin Already Loaded: " + p.Name)
				flag = true
			}
		}

		if ! flag {
	                plugins = append(plugins, tmpplg)
		}
        }

	if len(plugins) < 1 {
		Log("No Plugins Loaded: Do .so files exist in: " + plgpath + "?")
		fmt.Println("No Plugins Loaded: Do .so files exist in: " + plgpath + "?")
		return errors.New("No Plugins Loaded")
	} else {
		return nil
	}
}

func Do_Scrapes(c *Config) {
	var check Check

	if c.DefaultScrapeTime < 1 {
		c.DefaultScrapeTime = 300
	}

	for {
		for i := 0; i < len(c.Hosts); i++ {
			if c.Hosts[i].ScrapeTime == 0 {
				c.Hosts[i].ScrapeTime = c.DefaultScrapeTime
			}

			if len(c.Hosts[i].HostPaths) == 0 {
				c.Hosts[i].HostPaths = append(c.Hosts[i].HostPaths, "/checkandclear")
			}

			time.Sleep(time.Duration(c.Hosts[i].ScrapeTime) * time.Second)

			for _, hp := range c.Hosts[i].HostPaths {
				resp, err := http.Get("http://" + c.Hosts[i].HostName + hp)
				if err != nil {
					now := time.Now()
					current_time := time.Now().Local()
					epoch := now.Unix()
					t := current_time.Format("Jan 02 2006 03:04:05")

					check.Host = c.Hosts[i].HostName
					check.TimeStamp = t
					check.EpochTime = epoch
					check.Command = "scrape: " + c.Hosts[i].HostName + hp
					check.Output = "failed to scrape: " + err.Error()
					check.Retval = 1
					
					bytes, _ := json.Marshal(check)

					for _, failplg := range c.Hosts[i].FailurePlugins {
						for _, p := range plugins {
							if p.Name == failplg {
								_, _ = p.Function(string(bytes), false)
							}
						}
					}	

				} else {
					var chk []Check
					bytes, _ := ioutil.ReadAll(resp.Body)
					json.Unmarshal(bytes, &chk)
					psize := len(c.Hosts[i].Plugins)

					if psize < 1 {

					} else {
						for _, hstplg := range c.Hosts[i].Plugins {
							for _, p := range plugins {
								if p.Name == hstplg {
									_, _ = p.Function(string(bytes), true)
								}
							}
						}
					}
				}
			}
		}
	}
}

func main() {
	GetConfigs()
	for _, co := range configs {
		LoadPlugins(co.PluginPath)
	}

	for i := 0; i < len(configs); i++ {
		c := configs[i]
		go Do_Scrapes(&c)
	}

	router := mux.NewRouter()
	router.HandleFunc("/whoareyou", handleWhoAreYou)
	router.HandleFunc("/ping", handlePing)
	router.HandleFunc("/checks", handleChecks)
	router.HandleFunc("/checkandclear", handleCheckAndClear)
	router.HandleFunc("/statusof", handleStatusOf)

	err := http.ListenAndServe(":9051", router)
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
	}
}
