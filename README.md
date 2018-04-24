## fluentd2sls

`fluentd2sls` is a high performance service for relay and format fluentd log to `Aliyun Log Service(sls)`, you can custom parser by yourself.


## What you can do with it

1. Reduce k8s fluentd load, as I know json decode and `aliyun_sls_sdk` VERY slow in ruby code. The solution `fluentd` combine `aliyun_sls_sdk` [fluentd-pilot](https://github.com/AliyunContainerService/fluentd-pilot) @faecf52 write log to sls is not stable, when log more than 3000req/s fluentd eat 90%+ cpu. when log more than 200req/s it easier to be zombie when fluentd reload.
2. Custom your parser, by default use `DockerLogParser` for decode inner json in docker log.So you can write other parser for parser your custom log format, I will rewrite it by reflect. emmm...

## How it work
```
+----------------------+
|        fluentd       |
+----------------------+
            |
            | 1.send log in fluentd-syslog UDP mode
            | 
            |
+-----------v----------+-------------------------------------+
|    syslog service    |                         fluentd2sls |
+----------------------+                                     |
|           |                                                |
|           | 2.parser docker log format                     |
|           |                                                |
+-----------v----------+                                     |
|     syslog parser    |                                     |
+----------------------+                                     |
|           |                                                |
|           | 3.parser json format                           |
|           |                                                |
+-----------v----------+                                     |
|    DockerLogParser   |                                     |
+----------------------+                                     |
|           |                                                |
|           | 4.send to sls                                  |
|           |                                                |
+-----------v----------+                                     |
|   aliyun-log-go-sdk  |                                     |
+----------------------+-------------------------------------+
            |
            | 5.encrypt、encode、zip post HTTP
            |
+-----------v----------+
|  Aliyun Log Service  |
+----------------------+
```

## performance

|1Core 1Gi | fluentd combine aliyun_sls_sdk | fluentd2sls combine aliyun-sdk-golang |
| -------  |:--:| :--:|
| parser   | 1x | 4x  |
| sls sdk  | 1x | 18x |
| 1 deployment 1500req/s  | ~75% CPU | ~8% CPU |

## Try it
Install fluentd-pilot in your k8s cluster, set env like `192.168.50.78` replace fluentd2sls host.

```
        env:
          - name: "FLUENTD_OUTPUT"
            value: "syslog"
          - name: "SYSLOG_HOST"
            value: "192.168.50.78"
          - name: "SYSLOG_PORT"
            value: "233"

```

Prerequisites:

- ESC x1 with Linux x64
- Due performance reason DO NOT run in docker with production env.
- Create a ram account with full sls permission in aliyun console, get `AccessKeyID` and `AccessKeySecret`.
- Create a loghub project named `kubernetes`.
- Protocol tested with `RFC3164` (fluent-plugin-remote_syslog v 0.3.3) and `RFC5424` (fluent-plugin-remote_syslog-5424 v 0.1.1), `RFC3164` message must less than 1024byte.

```
git clone https://github.com/mengskysama/fluentd2sls

edit fluentd-pilot config.yml
sls:
  Name: "kubernetes"
  Endpoint: "cn-hangzhou-vpc.log.aliyuncs.com"
  AccessKeyID: "RAM AK"
  AccessKeySecret: "RAM SK"

relay:
  BindAddr: "0.0.0.0:233"
  Protocol: "RFC3164"
  LogLevel: "DEBUG"

docker-compose build
docker-compose up

Config nginx deployment log to stdout with json format
ln -sf /dev/stdout /var/log/nginx/access.log

edit nginx.conf

    log_format mixed    '{"remote_addr":"$remote_addr", "local_time":"$time_local",'
                        '"request":"$request","status":"$status","body_bytes_sent":"$body_bytes_sent",'
                        '"http_referer":"$http_referer","http_user_agent":"$http_user_agent",'
                        '"host":"$host","request_time":"$request_time"}';
    access_log   /var/log/nginx/access.log mixed;

note: DockerLogParser will parser local_time from nginx time_local format
In common you can get a new Logstore deployment in loghub kubernetes.
```


## Related projects

- [fluentd-pilot](https://github.com/AliyunContainerService/fluentd-pilot): Collect logs in docker containers
- [json-iterator/go](https://github.com/json-iterator/go): A high-performance 100% compatible drop-in replacement of "encoding/json"
- [aliyun-log-go-sdk](github.com/aliyun/aliyun-log-go-sdk): go loghub sdk
- [go-syslog.v2](github.com/aliyun/aliyun-log-go-sdk): Syslog server library for go, build easy your custom syslog server over UDP, TCP or Unix sockets using RFC3164, RFC5424 and RFC6587
- [log4go](github.com/thinkboy/log4go): github.com/thinkboy/log4go

## Contribute

You are welcome to new issues and PR.

## License
MIT

