package main

import (
	"fmt"
	"strings"
	"strconv"
	"bytes"
        "gopkg.in/yaml.v2"
	"io/ioutil"
	"net/smtp"
	"encoding/json"
)

type Config struct {
	AlertList []string `yaml:"AlertList"`
	SMTPServer string `yaml:"SMTPServer"`
	FromAddress string `yaml:"FromAddress"`
	Subject string `yaml:"Subject"`
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

var config Config

func SendSMTPMessage(mailserver string, from string, to []string, subject string, body string) error {
	connection, err := smtp.Dial(mailserver)
	if err != nil {
		return err
	}
	defer connection.Close()

	connection.Mail(from)

	for _, addr := range to {
		connection.Rcpt(addr)
	}

	wc, err := connection.Data()
	if err != nil {
		return err
	}
	defer wc.Close()

	combined_to := strings.Join(to, ";")

	body = "To: " + combined_to + "\r\nFrom: " + from + "\r\nSubject: " + subject + "\r\n\r\n" + body

	buf := bytes.NewBufferString(body)
	_, err = buf.WriteTo(wc)
	if err != nil {
		return err
	}

	return nil
}

func Handle(check string, failed bool) (bool, error) {

	var chk Check
	b, err := ioutil.ReadFile("/etc/heimdall/plugins.d/alert.yml")
	if err != nil {
		return false, err
	}

	yml := string(b)
	err = yaml.Unmarshal([]byte(yml), &config)

	if err != nil {
		return false, err
	}

	_ = json.Unmarshal([]byte(check), &chk)

	body := "Host: " + chk.Host + "\n"
	body += "TimeStamp: " + chk.TimeStamp + "\n"
	body += "EpochTime: " + strconv.FormatInt(chk.EpochTime, 10) + "\n"
	body += "Command: " + chk.Command + "\n"
	body += "Output: " + chk.Output + "\n"
	body += "Retval: " + strconv.Itoa(chk.Retval) + "\n"

	err = SendSMTPMessage(config.SMTPServer, config.FromAddress, config.AlertList, config.Subject, body)

	fmt.Printf("%+v\n", config)
	fmt.Println(check)
	fmt.Println(body)

	if err != nil {
		fmt.Println("failed to send alert email")
		return false, err		
	}

	return true, nil

}

var PluginName = "Alert_SMTP"
var PluginVersion = "0.1"
