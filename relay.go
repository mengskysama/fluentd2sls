package main

import (
	"time"

	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/gogo/protobuf/proto"
	log "github.com/thinkboy/log4go"
)

const RELAY_BUFF_MAX = 16384

// limit 5MB each LogGroup
const RELAY_FLUSH_BATCH = 4096

// flush message to sls interval
const RELAY_FLUSH_INTERVAL = 10

// time of n*RELAY_FLUSH_INTERVAL realy not recv any message
const RELAY_TIMEOIT_TIMES = 3

// SyslogMessage ...
type SyslogMessage struct {
	Log             string
	Stream          string
	Host            string
	Target          string `json:"@target"`
	DockerContainer string `json:"docker_container"`
	Pod             string `json:"k8s_pod"`
}

// Relay struct
type Relay struct {
	logstoreName   string
	recvFromSysLog chan *SyslogMessage
	recvClosing    chan bool
	stage          int
	logstore       *sls.LogStore
}

// NewRelay Create logstore if not exist
func NewRelay(logstoreName string) (relay *Relay, err error) {
	stage := 1
	var logstore *sls.LogStore
	logstore, err = project.GetLogStore(logstoreName)
	if err != nil {
		// May not exists
	}
	if logstore == nil {
		stage = 0
		err = project.CreateLogStore(logstoreName, 1, 2)
		if err != nil {
			log.Debug("call CreateLogStore %s fail.", logstoreName)
			return
		}
		log.Debug("CreateLogStore %s success.", logstoreName)
		logstore, err = project.GetLogStore(logstoreName)
		// TODO:
		// Creating few seconds couldn't write
	}
	relay = &Relay{
		logstoreName:   logstoreName,
		recvFromSysLog: make(chan *SyslogMessage, RELAY_BUFF_MAX),
		recvClosing:    make(chan bool),
		stage:          stage,
		logstore:       logstore,
	}
	go relay.run()
	return
}

func (r *Relay) run() {
	t := int64(0)
	timeoutTimes := 0
	timer := time.NewTimer(time.Second * RELAY_FLUSH_BATCH)
	for {
		idx := 0
		t = time.Now().Unix()
		messages := [RELAY_FLUSH_BATCH]*SyslogMessage{}
		for {
			timer.Reset(time.Second * time.Duration(RELAY_FLUSH_INTERVAL))
			select {
			case messages[idx] = <-r.recvFromSysLog:
				timer.Stop()
				timeoutTimes = 0
				idx++
			case <-r.recvClosing:
				timer.Stop()
				log.Debug("relay %s closed.", r.logstoreName)
				return
			case <-timer.C:
				break
			}
			if idx == 0 {
				timeoutTimes++
				if timeoutTimes > RELAY_TIMEOIT_TIMES {
					log.Debug("relay %s not recived message closed.", r.logstoreName)
					UnRegisterRelay(r.logstoreName)
				}
				t = time.Now().Unix()
			} else if idx == RELAY_FLUSH_BATCH || time.Now().Unix()-t > RELAY_FLUSH_INTERVAL {
				goto WRITE_TO_SLS
			}
		}
	WRITE_TO_SLS:
		r.writeLogToSls(messages[:idx])
	}
}

func (r *Relay) writeLogToSls(syslogMessages []*SyslogMessage) (err error) {

	var p FluentdParser = &DockerLogParser{}
	logs := make([]*sls.Log, len(syslogMessages))

	for idx := range syslogMessages {
		l, err := p.Dump(syslogMessages[idx])
		if err != nil {
			log.Debug("[%s] Dump message %v fail", r.logstoreName, syslogMessages[idx])
			continue
		}
		logs[idx] = l
	}

	loggroup := &sls.LogGroup{
		Topic:  proto.String(r.logstoreName),
		Source: proto.String(hostName),
		Logs:   logs,
	}

	err = r.logstore.PutLogs(loggroup)
	if err != nil {
		log.Debug("PutLogs [%s] fail, err: %s", r.logstoreName, err)
		return
	}

	log.Debug("PutLogs [%s] success cnt %d", r.logstoreName, len(syslogMessages))
	return
}
