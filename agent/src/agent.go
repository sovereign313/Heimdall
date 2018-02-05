package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"os"
	"time"
	"worker"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Label       string   `yaml:"Label"`
	CommandType string   `yaml:"CommandType"`
	Command     string   `yaml:"Command"`
	CheckFreq   int      `yaml:"CheckFreq"`
	Params      []string `yaml:"Params"`
	Enabled     bool     `yaml:"Enabled"`
}

var configs []Config
var checks []worker.Check

func MakeSkel() error {
	err := os.MkdirAll("/etc/heimdall/config.d", 0644)
	if err != nil {
		return err
	}

	file, err := os.OpenFile("/etc/heimdall/config.d/cpu.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString("Label: Load Average\n")
	file.WriteString("CommandType: internal\n")
	file.WriteString("Command: LoadAverage\n")
	file.WriteString("CheckFreq: 60\n")
	file.WriteString("Enabled: true\n")
	file.Close()

	file, err = os.OpenFile("/etc/heimdall/config.d/memory.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString("Label: Memory Usage\n")
	file.WriteString("CommandType: internal\n")
	file.WriteString("Command: MemUsage\n")
	file.WriteString("CheckFreq: 60\n")
	file.WriteString("Enabled: true\n")
	file.Close()

	file, err = os.OpenFile("/etc/heimdall/config.d/disks.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString("Label: Disk usage\n")
	file.WriteString("CommandType: internal\n")
	file.WriteString("Command: CheckDiskUsage\n")
	file.WriteString("CheckFreq: 60\n")
	file.WriteString("Params: [\"/\", \"/home\", \"/proj/app\", \"/tmp\", \"/var\"]\n")
	file.WriteString("Enabled: true\n")
	file.Close()

	file, err = os.OpenFile("/etc/heimdall/config.d/rootpasswdexp.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString("Label: Root Password Expiration\n")
	file.WriteString("CommandType: internal\n")
	file.WriteString("Command: CheckPassword\n")
	file.WriteString("CheckFreq: 60\n")
	file.WriteString("Params: [\"root\"]\n")
	file.WriteString("Enabled: true\n")
	file.Close()

	file, err = os.OpenFile("/etc/heimdall/config.d/oraclepasswdexp.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString("Label: Oracle Password Expiration\n")
	file.WriteString("CommandType: internal\n")
	file.WriteString("Command: CheckPassword\n")
	file.WriteString("CheckFreq: 60\n")
	file.WriteString("Params: [\"oracle\"]\n")
	file.WriteString("Enabled: true\n")
	file.Close()

	file, err = os.OpenFile("/etc/heimdall/config.d/ssh.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString("Label: Check SSH\n")
	file.WriteString("CommandType: internal\n")
	file.WriteString("Command: CheckSSH\n")
	file.WriteString("CheckFreq: 60\n")
	file.WriteString("Enabled: true\n")
	file.Close()

	file, err = os.OpenFile("/etc/heimdall/config.d/swap.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString("Label: Swap Usage\n")
	file.WriteString("CommandType: internal\n")
	file.WriteString("Command: CheckSwap\n")
	file.WriteString("CheckFreq: 60\n")
	file.WriteString("Enabled: true\n")
	file.Close()

	file, err = os.OpenFile("/etc/heimdall/config.d/inode.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString("Label: Inode Usage\n")
	file.WriteString("CommandType: internal\n")
	file.WriteString("Command: CheckInodes\n")
	file.WriteString("CheckFreq: 60\n")
	file.WriteString("Enabled: true\n")
	file.Close()

	file, err = os.OpenFile("/etc/heimdall/config.d/ntp.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString("Label: NTP Skew\n")
	file.WriteString("CommandType: internal\n")
	file.WriteString("Command: CheckNTPSkew\n")
	file.WriteString("CheckFreq: 60\n")
	file.WriteString("Params: [\"pool.ntp.org\"]\n")
	file.WriteString("Enabled: true\n")
	file.Close()

	file, err = os.OpenFile("/etc/heimdall/config.d/mailq.yml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString("Label: MailQ Count\n")
	file.WriteString("CommandType: internal\n")
	file.WriteString("Command: CheckMailQ\n")
	file.WriteString("CheckFreq: 60\n")
	file.WriteString("Enabled: true\n")
	file.Close()

	return nil
}

func Log(message string) {
	file, err := os.OpenFile("./heimdall.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed To Open Log File: " + err.Error())
	}
	file.Close()

	current_time := time.Now().Local()
	t := current_time.Format("Jan 02 2006 03:04:05")
	_, err = file.WriteString(t + " - Heimdall: " + message + "\n")
}

func GetConfigs() {

	_, err := os.Stat("/etc/heimdall/config.d/")
	if os.IsNotExist(err) {
		err := MakeSkel()
		if err != nil {
			fmt.Println("Error Setting Up /etc/heimdall/config.d/ and default settings file: " + err.Error())
			Log("Error Setting Up /etc/heimdall/config.d/ and default settings file: " + err.Error())
			return
		}
	}

	files, err := ioutil.ReadDir("/etc/heimdall/config.d/")
	if err != nil {
		fmt.Println("Error Reading /etc/heimdall/config.d/: " + err.Error())
		Log("Error Reading /etc/heimdall/config.d/: " + err.Error())
		return
	}

	if len(files) < 1 {
		fmt.Println("/etc/heimdall/config.d/ exists, but is empty. No Configs Loaded")
		Log("/etc/heimdall/config.d/ exists, but is empty. No Configs Loaded")
	}

	for _, f := range files {
		b, err := ioutil.ReadFile("/etc/heimdall/config.d/" + f.Name())
		if err != nil {
			fmt.Println("Error Opening File: /etc/heimdall/config.d/" + f.Name() + ": " + err.Error())
			Log("Error Opening File: /etc/heimdall/config.d/" + f.Name() + ": " + err.Error())
		}

		yml := string(b)

		c := Config{}
		err = yaml.Unmarshal([]byte(yml), &c)

		if err != nil {
			fmt.Println("Couldn't Parse YAML File /etc/heimdall/config.d/" + f.Name() + ": " + err.Error())
			Log("Couldn't Parse YAML File /etc/heimdall/config.d/" + f.Name() + ": " + err.Error())
		}

		configs = append(configs, c)
	}

}

func handleWhoAreYou(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Heimdall Agent")
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
	retvals := []worker.Check{}
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

func Do_Checks(c *Config, chanl chan worker.Check) {
	var check worker.Check

	if c.CheckFreq < 1 {
		c.CheckFreq = 60
	}

	for {
		time.Sleep(time.Duration(c.CheckFreq) * time.Second)
		if c.CommandType == "internal" {
			if c.Command == "LoadAverage" {
				check, _ = worker.LoadAverage(c.Label)
			} else if c.Command == "MemUsage" {
				check, _ = worker.MemUsage(c.Label)
			} else if c.Command == "CheckSSH" {
				check, _ = worker.CheckSSH(c.Label)
			} else if c.Command == "CheckSwap" {
				check, _ = worker.CheckSwap(c.Label)
			} else if c.Command == "CheckMailQ" {
				check, _ = worker.CheckMailQ(c.Label)
			} else if c.Command == "CheckDiskUsage" {
				for _, dir := range c.Params {
					check, _ = worker.CheckDiskUsage(c.Label, dir)
				}
			} else if c.Command == "CheckPassword" {
				for _, user := range c.Params {
					check, _ = worker.CheckPassword(c.Label, user)
				}
			} else if c.Command == "CheckNTPSkew" {
				for _, ntpserver := range c.Params {
					check, _ = worker.CheckNTPSkew(c.Label, ntpserver)
				}
			}
		} else {
			check, _ = worker.RunExternal(c.Label, c.Command)
		}

		hstname, err := os.Hostname()
		if err != nil {
			check.Host = "Error Getting Hostname: " + err.Error()
		} else {
			check.Host = hstname
		}

		chanl<-check
	}
}

func main() {
	GetConfigs()

	go func() {

		chanl := make(chan worker.Check)
		for i := 0; i < len(configs); i++ {
			c := configs[i]
			if c.Enabled {
				go Do_Checks(&c, chanl)
			}
		}

		for {
			tmp := <-chanl
			checks = append(checks, tmp)
		}
		
	}()

	router := mux.NewRouter()
	router.HandleFunc("/whoareyou", handleWhoAreYou)
	router.HandleFunc("/ping", handlePing)
	router.HandleFunc("/checks", handleChecks)
	router.HandleFunc("/checkandclear", handleCheckAndClear)
	router.HandleFunc("/statusof", handleStatusOf)

	err := http.ListenAndServe(":9050", router)
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
	}
}
