package main

import (
    "fmt"
    "github.com/confluentinc/confluent-kafka-go/kafka"
    "github.com/golang/protobuf/proto"
    conf "github.com/woodliu/prometheusWriter/pkg/config"

    "github.com/woodliu/prometheusWriter/pkg/prompb"
    "log"
    "strings"
    "time"
)

// 本功能仅用于测试使用
func main() {
    cfg, err := conf.Load("../config.json")
    if err != nil {
       log.Fatal(err)
    }

    p, err := kafka.NewProducer(&kafka.ConfigMap{
       // 设置接入点，请通过控制台获取对应Topic的接入点。
       "bootstrap.servers": strings.Join(cfg.Kafka.Servers, ","),
       // SASL 验证机制类型默认选用 PLAIN
       "sasl.mechanism": "PLAIN",
       // 在本地配置 ACL 策略。
       "security.protocol": "SASL_PLAINTEXT",
       // username 是实例 ID + # + 配置的用户名，password 是配置的用户密码。
       "sasl.username": fmt.Sprintf("%s#%s", cfg.Kafka.Auth.InstanceID, cfg.Kafka.Auth.UserName),
       "sasl.password": cfg.Kafka.Auth.Password,

       // Kafka producer 的 ack 有 3 种机制，分别说明如下：
       // -1 或 all：Broker 在 leader 收到数据并同步给所有 ISR 中的 follower 后，才应答给 Producer 继续发送下一条（批）消息。
       // 这种配置提供了最高的数据可靠性，只要有一个已同步的副本存活就不会有消息丢失。注意：这种配置不能确保所有的副本读写入该数据才返回，
       // 可以配合 Topic 级别参数 min.insync.replicas 使用。
       // 0：生产者不等待来自 broker 同步完成的确认，继续发送下一条（批）消息。这种配置生产性能最高，但数据可靠性最低（当服务器故障时可能会有数据丢失，如果 leader 已死但是 producer 不知情，则 broker 收不到消息）
       // 1： 生产者在 leader 已成功收到的数据并得到确认后再发送下一条（批）消息。这种配置是在生产吞吐和数据可靠性之间的权衡（如果leader已死但是尚未复制，则消息可能丢失）
       // 用户不显示配置时，默认值为1。用户根据自己的业务情况进行设置
       "acks": 1,
       // 请求发生错误时重试次数，建议将该值设置为大于0，失败重试最大程度保证消息不丢失
       "retries": 0,
       // 发送请求失败时到下一次重试请求之间的时间
       "retry.backoff.ms": 100,
       // producer 网络请求的超时时间。
       "socket.timeout.ms": 6000,
       // 设置客户端内部重试间隔。
       "reconnect.backoff.max.ms": 3000,
    })
    if err != nil {
       log.Fatal(err)
    }

    defer p.Close()

    // 产生的消息 传递至报告处理程序
    go func() {
       for e := range p.Events() {
           switch ev := e.(type) {
           case *kafka.Message:
               if ev.TopicPartition.Error != nil {
                   fmt.Printf("Delivery failed: %v\n", ev.TopicPartition)
               } else {
                   fmt.Printf("Delivered message to %v\n", ev.TopicPartition)
               }
           }
       }
    }()

    wr := prompb.WriteRequest{
        Timeseries: []prompb.TimeSeries{
            {
                Labels: []prompb.Label{
                    {
                        Name: "__name__",
                        Value: "test_kafka111",
                    },
                    {
                        Name: "testlabel11",
                        Value: "test",
                    },
                },
                Samples: []prompb.Sample{
                    {
                        Timestamp: time.Now().Unix(),
                        Value: 1.1,
                    },
                },
                Exemplars: []prompb.Exemplar{
                    {
                        Labels: []prompb.Label{
                            {
                                Name: "traceId",
                                Value: "testTraceId",
                            },
                        },
                        Value: 1.2,
                        Timestamp: time.Now().Unix(),
                    },
                },
            },
        },
    }

    topic := cfg.Kafka.Topics
    data,err := proto.Marshal(&wr)
    if nil != err{
        fmt.Println(err)
    }

    err = p.Produce(&kafka.Message{
        TopicPartition: kafka.TopicPartition{Topic: &topic[0], Partition: kafka.PartitionAny},
        Value:          data,
    }, nil)
    if nil != err{
        fmt.Println(err)
    }

    wr.Timeseries[0].Labels[0].Value = "test_kafka222"
    data,err = proto.Marshal(&wr)
    if nil != err{
        fmt.Println(err)
    }

    err = p.Produce(&kafka.Message{
        TopicPartition: kafka.TopicPartition{Topic: &topic[0], Partition: kafka.PartitionAny},
        Value:          data,
    }, nil)
    if nil != err{
        fmt.Println(err)
    }

    // 等待消息传递
    p.Flush(10 * 1000)
}