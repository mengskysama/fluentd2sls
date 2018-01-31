package main

import (
	"errors"
	"flag"

	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/json-iterator/go"
	log "github.com/thinkboy/log4go"
	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Config of app
type Config struct {
	SLS struct {
		Name            string `yaml:"Name"`
		Endpoint        string `yaml:"Endpoint"`
		AccessKeyID     string `yaml:"AccessKeyID"`
		AccessKeySecret string `yaml:"AccessKeySecret"`
	}
	Relay struct {
		BindAddr string `yaml:"BindAddr"`
	}
}

var project *sls.LogProject
var config *Config
var relays map[string]*Relay
var relayLock sync.Mutex
var hostName, _ = os.Hostname()

// LoadConfig ...
func LoadConfig(filePath string) (err error) {
	var dat []byte
	if dat, err = ioutil.ReadFile(filePath); err != nil {
		log.Error("Config file read err %s", err)
		return
	}
	config = &Config{}
	if err = yaml.Unmarshal([]byte(dat), config); err != nil {
		log.Error("Config file paser err %s", err)
		return
	}
	project = &sls.LogProject{
		Name:            config.SLS.Name,
		Endpoint:        config.SLS.Endpoint,
		AccessKeyID:     config.SLS.AccessKeyID,
		AccessKeySecret: config.SLS.AccessKeySecret,
	}
	return
}

// RegisterRelay Register and return a relay
func RegisterRelay(logstoreName string) (relay *Relay, err error) {
	relayLock.Lock()
	defer relayLock.Unlock()
	exists := false
	relay, exists = relays[logstoreName]
	if !exists {
		relay, err = NewRelay(logstoreName)
		if err != nil {
			log.Warn("relay %s create fail, %s", logstoreName, err)
			return
		}
		relays[logstoreName] = relay
		log.Debug("revice %s register relay success", logstoreName)
	}
	return
}

// UnRegisterRelay Register and return a relay
func UnRegisterRelay(logstoreName string) (err error) {
	relayLock.Lock()
	defer relayLock.Unlock()
	relay, exists := relays[logstoreName]
	if !exists {
		log.Warn("relay %s UnRegisterRelay fail not exist", logstoreName)
		return
	}
	close(relay.recvClosing)
	delete(relays, logstoreName)
	return
}

// ProcessSysLog for recv data from syslog
func ProcessSysLog(data string) (err error) {
	// data := `2017-12-07T04:17:28Z	fluentd-pilot	{"log":"{\"type\": \"access\", \"remote_addr\": \"::ffff:172.16.6.49\", \"time\":\"2017-12-07T04:17:28.922Z\", \"method\": \"GET\", \"uri\": \"/health\", \"version\": \"1.1\", \"status\": 200, \"length\": 25, \"referrer\": \"-\", \"user-agent\": \"stress 1.0\", \"request_time\": 0.130}\n","stream":"stdout","@timestamp":"2017-12-07T04:17:28.951","host":"fluentd-pilot-s85mg","@target":"k8skoatemplate","docker_container":"k8s_k8s-koa-template_k8s-koa-template-7c749b67fd-tzztz_default_81f0941f-d98a-11e7-9d87-00163e0da962_0","k8s_pod":"k8s-koa-template-7c749b67fd-tzztz"}`)
	p1 := strings.Index(data, "\t")
	s := data[p1+1:]
	p2 := strings.Index(s, "\t")
	s = s[p2+1:]

	msg := &SyslogMessage{}
	err = jsoniter.Unmarshal([]byte(s), &msg)
	if err != nil {
		return errors.New("message is not json")
	}

	relay, err := RegisterRelay(msg.Target)
	if err != nil {
		return errors.New("RegisterRelay failed")
	}

	select {
	case relay.recvFromSysLog <- msg:
		return
	case <-time.After(time.Millisecond * time.Duration(100)):
		return log.Error("relay %s buffer is full?", msg.Target)
	}
}

func init() {
	relays = make(map[string]*Relay)
}

func main() {
	defer time.Sleep(time.Second * 1)

	log.Debug("fluentd2sls v0.0.1")
	flagConfigPath := flag.String("f", "config.yml", "config path")
	flag.Parse()
	if err := LoadConfig(*flagConfigPath); err != nil {
		return
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)
	server := syslog.NewServer()
	server.SetFormat(syslog.RFC3164)
	server.SetHandler(handler)
	if err := server.ListenUDP(config.Relay.BindAddr); err != nil {
		log.Error("%s", err)
		return
	}
	server.Boot()

	go func(channel syslog.LogPartsChannel) {
		// TODO
		// coroutine * NumCPU
		for logParts := range channel {
			err := ProcessSysLog(logParts["content"].(string))
			if err != nil {
				log.Warn("process syslog error %s", err)
			}
		}
	}(channel)

	server.Wait()
}
