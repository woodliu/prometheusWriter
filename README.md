### 功能支持列表

- 支持从kafka消费Prometheus指标数据，数据使用protobuf编码
- 支持Prometheus exemplar功能
- 支持exemplar的wal
- 支持remote write指标到存储

### kafka消费端

> 本项目使用的是腾讯的cKafka

golang的kafka消费端需要用到`github.com/confluentinc/confluent-kafka-go/kafka`，使用该库之前需要[安装](https://github.com/confluentinc/confluent-kafka-go#installing-librdkafka)`librdkafka`库，但**不支持在Windows系统**上安装`librdkafka`。安装步骤如下：

```
git clone https://github.com/edenhill/librdkafka.gitcd librdkafka./configuremakesudo make install
```

> 环境上运行时可以考虑将`librdkafka`库编译到镜像中。如使用Alpine镜像时执行`apk add librdkafka-dev pkgconf`安装即可。[官方文档](https://github.com/confluentinc/confluent-kafka-go#using-go-modules)中有提到，如果使用Alpine Linux ，编译方式为：go build -tags musl ./...

### Metrics的写入

只需将metrics使用`proto.Marshal`编码到`promWR`即可:

```golang
func (c *client) WriteRaw(    ctx context.Context,    promWR []byte,    opts WriteOptions,) (WriteResult, WriteError) {    var result WriteResult    encoded := snappy.Encode(nil, promWR)    body := bytes.NewReader(encoded)    req, err := http.NewRequest("POST", c.writeURL, body)    if err != nil {        return result, writeError{err: err}    }    req.Header.Set("Content-Type", "application/x-protobuf")    req.Header.Set("Content-Encoding", "snappy")    req.Header.Set("User-Agent", c.userAgent)    req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")    if opts.Headers != nil {        for k, v := range opts.Headers {            req.Header.Set(k, v)        }    }    resp, err := c.httpClient.Do(req.WithContext(ctx))    if err != nil {        return result, writeError{err: err}    }    result.StatusCode = resp.StatusCode    defer resp.Body.Close()    if result.StatusCode/100 != 2 {        writeErr := writeError{            err:  fmt.Errorf("expected HTTP 200 status code: actual=%d", resp.StatusCode),            code: result.StatusCode,        }        body, err := ioutil.ReadAll(resp.Body)        if err != nil {            writeErr.err = fmt.Errorf("%v, body_read_error=%s", writeErr.err, err)            return result, writeErr        }        writeErr.err = fmt.Errorf("%v, body=%s", writeErr.err, body)        return result, writeErr    }    return result, nil}
```

### metric的查询

使用victoriametrics时，强烈建议同时部署grafana，使用grafana中的`Explore`功能来查找metrics。victoriametrics的vmselect组件自带的UI很不方便。

### 镜像编译

如上所述，如果需要在需要Alpine Linux中进行编译，则需要在在Dockerfile中添加如下内容：

```
RUN apk add git && apk add librdkafka-dev pkgconf && apk add build-base && apk add alpine-sdk
```

由于上述lib的安装比较满，为了加快安装，可以将安装了这些lib的镜像作为基础镜像。

```dockerfile
FROM golang:1.16.8-alpine3.14 as buildWORKDIR /appRUN apk add git && apk add librdkafka-dev pkgconf && apk add build-base && apk add alpine-sdkENV http_proxy= GO111MODULE=on GOPROXY=https://goproxy.cn,direct GOPRIVATE=*.weimob.comCOPY go.mod .COPY go.sum .COPY . .RUN cd cmd/ && GOOS=linux go build -tags musl -o ../prometheusWriter main.goCMD ["/app/prometheusWriter"]FROM alpine:latestWORKDIR /appRUN sed -i s/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g /etc/apk/repositoriesRUN apk add tzdataRUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo "Asia/Shanghai" > /etc/timezoneCOPY --from=build /app/prometheusWriter /app/RUN chmod +x /app/prometheusWriterCOPY config.json /app/config.jsonCMD ["/app/prometheusWriter"]
```

### 支持Exemplar

Exemplar的数据结构比较简单，就是个ring buffer。

下面是使用curl命令进行查找的例子：

```shell
# curl  '127.0.0.1:8000' --header 'Content-Type: application/json' -d '{"start":"1632980302","end":"1632980402","query":"{testlabel11=\"test\"}"}' 
```

