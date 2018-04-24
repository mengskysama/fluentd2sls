FROM golang:alpine3.6

RUN printf "http://mirrors.aliyun.com/alpine/v3.6/community\nhttp://mirrors.aliyun.com/alpine/v3.6/main\n" > /etc/apk/repositories  && \
  apk add --update --no-cache git && \
  go get gopkg.in/mcuadros/go-syslog.v2 && \
  go get github.com/aliyun/aliyun-log-go-sdk && \
  go get github.com/json-iterator/go && \
  go get gopkg.in/yaml.v2 && \
  go get github.com/gogo/protobuf/proto && \
  go get github.com/thinkboy/log4go && \
  apk del --purge git

WORKDIR /opt/fluentd2sls
COPY . .
RUN rm config.yml && go build -o fluentd2sls
