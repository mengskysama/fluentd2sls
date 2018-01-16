package main

import (
	"fmt"
	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/gogo/protobuf/proto"
	"github.com/json-iterator/go"
	log "github.com/thinkboy/log4go"
	"regexp"
	"time"
)

/*
	stage = 0 init
	stage = 1 ready
	stage = 2 closing
*/

// FluentdParser interface
type FluentdParser interface {
	Dump(*SyslogMessage) (*sls.Log, error)
}

// DockerLogParser is one of FluentdParser
type DockerLogParser struct {
	name string
}

const ngxTimeFmt = "02/Jan/2006:15:04:05 -0700"

var regIntFormat = regexp.MustCompile(`(\.0{2,})$`)

// Dump parser docker log to sls message
func (p *DockerLogParser) Dump(msg *SyslogMessage) (l *sls.Log, err error) {
	var dat map[string]interface{}
	err = jsoniter.Unmarshal([]byte(msg.Log), &dat)
	if err != nil {
		// If log field is not vaild json output direct
		dat = make(map[string]interface{}, 1)
		dat["log"] = msg.Log
		err = nil
	}

	logTime := time.Now()
	// try parser ngx local_time
	localTime, exists := dat["local_time"]
	if exists {
		v := localTime.(string)
		t, err := time.Parse(ngxTimeFmt, v)
		if err != nil {
			logTime = t
		}
	}

	contents := make([]*sls.LogContent, len(dat)+3)
	idx := 0

	for k, v := range dat {
		content := &sls.LogContent{
			Key: proto.String(k),
		}

		switch v.(type) {
		case string:
			content.Value = proto.String(v.(string))
		case float64:
			t := fmt.Sprintf("%f", v.(float64))
			content.Value = proto.String(regIntFormat.ReplaceAllString(t, "${2}"))
		case []interface{}:
			b, err := jsoniter.Marshal(&v)
			if err != nil {
				content.Value = proto.String("")
			} else {
				content.Value = proto.String(string(b))
			}
		default:
			log.Warn("unkown value type")
			content.Value = proto.String("")
		}

		contents[idx] = content
		idx++
	}

	contents[idx] = &sls.LogContent{
		Key:   proto.String("stream"),
		Value: proto.String(msg.Stream),
	}
	idx++
	contents[idx] = &sls.LogContent{
		Key:   proto.String("container"),
		Value: proto.String(msg.DockerContainer),
	}
	idx++
	contents[idx] = &sls.LogContent{
		Key:   proto.String("pod"),
		Value: proto.String(msg.Pod),
	}

	l = &sls.Log{
		Time:     proto.Uint32(uint32(logTime.Unix())),
		Contents: contents,
	}
	return
}
