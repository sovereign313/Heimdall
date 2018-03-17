package main

import (
	"os"
	"fmt"
	"strconv"
        "gopkg.in/yaml.v2"
	"io/ioutil"
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

type Config struct {
	SNSRegion string `yaml:"SNSRegion"`
	SNSTopicARN string `yaml:"SNSTopicARN"`
	AWSAccessKey string `yaml:"AWS_ACCESS_KEY_ID"`
	AWSSecretKey string `yaml:"AWS_SECRET_ACCESS_KEY"`
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

func SendSNSMessage(message string, topicarn string, snsregion string) error {
       svc := sns.New(session.New(&aws.Config{Region: aws.String(snsregion)}))

        params := &sns.PublishInput{
                Message: aws.String(message),
                TopicArn: aws.String(topicarn),
        }

        _, err := svc.Publish(params)  

        if err != nil {                    
                return err
        }

	return nil
}

func Handle(check string, failed bool) (string, error) {

	var chk Check
	b, err := ioutil.ReadFile("/etc/heimdall/plugins.d/alert_sns.yml")
	if err != nil {
		return "", err
	}

	yml := string(b)
	err = yaml.Unmarshal([]byte(yml), &config)

	if err != nil {
		return "", err
	}

	_ = json.Unmarshal([]byte(check), &chk)

	os.Setenv("AWS_ACCESS_KEY_ID", config.AWSAccessKey)
	os.Setenv("AWS_SECRET_ACCESS_KEY", config.AWSSecretKey)

	body := "Host: " + chk.Host + "\n"
	body += "TimeStamp: " + chk.TimeStamp + "\n"
	body += "EpochTime: " + strconv.FormatInt(chk.EpochTime, 10) + "\n"
	body += "Command: " + chk.Command + "\n"
	body += "Output: " + chk.Output + "\n"
	body += "Retval: " + strconv.Itoa(chk.Retval) + "\n"

	err = SendSNSMessage(body, config.SNSTopicARN, config.SNSRegion)

	if err != nil {
		fmt.Println("failed to send SNS Alert")
		return "", err		
	}

	return "success", nil

}

var PluginName = "Alert_SNS"
var PluginVersion = "0.1"
