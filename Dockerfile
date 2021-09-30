FROM harbor-devops.internal.hsmob.com/devops-dev/golang:1.16.8-alpine3.14 as build
WORKDIR /app
#RUN apk add git && apk add librdkafka-dev pkgconf && apk add build-base && apk add alpine-sdk
RUN git config --global --add url."https://weiqing.wang:v7E9PNf7xPwUTwuAevXa@stash.weimob.com/".insteadOf "ssh://git@stash.weimob.com/"
RUN git config --global --add url."https://weiqing.wang:v7E9PNf7xPwUTwuAevXa@stash.weimob.com/".insteadOf "http://stash.weimob.com/"
RUN git config --global --add url."https://weiqing.wang:v7E9PNf7xPwUTwuAevXa@stash.weimob.com/".insteadOf "https://stash.weimob.com/"

ENV http_proxy= GO111MODULE=on GOPROXY=https://goproxy.cn,direct GOPRIVATE=*.weimob.com

COPY go.mod .
COPY go.sum .
COPY . .

RUN cd cmd/ && GOOS=linux go build -tags musl -o ../prometheusWriter main.go

CMD ["/app/prometheusWriter"]

FROM alpine:latest
WORKDIR /app
RUN sed -i s/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g /etc/apk/repositories
RUN apk add tzdata
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo "Asia/Shanghai" > /etc/timezone
COPY --from=build /app/prometheusWriter /app/
RUN chmod +x /app/prometheusWriter
COPY config.json /app/config.json
CMD ["/app/prometheusWriter"]
