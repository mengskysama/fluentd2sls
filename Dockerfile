FROM golang:alpine3.6

RUN printf "http://mirrors.aliyun.com/alpine/v3.6/community\nhttp://mirrors.aliyun.com/alpine/v3.6/main\n" > /etc/apk/repositories  && \
  apk add --update git gcc musl-dev && \
  go get gopkg.in/mcuadros/go-syslog.v2 && \
  go get github.com/aliyun/aliyun-log-go-sdk && \
  go get github.com/thinkboy/log4go && \
  go get github.com/hungys/go-lz4 && \
  go get github.com/json-iterator/go && \
  go get gopkg.in/yaml.v2 && \
  apk del --purge git

WORKDIR /opt/fluentd2sls
COPY . .
RUN rm config.yml && go build -o fluentd2sls
