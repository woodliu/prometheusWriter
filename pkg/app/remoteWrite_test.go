package app

import (
    "context"
    "fmt"
    "github.com/golang/protobuf/proto"
    conf "github.com/woodliu/prometheusWriter/pkg/config"
    "github.com/woodliu/prometheusWriter/pkg/lib"
    "log"
    "testing"
    "time"
)

func TestRemoteWrite(t *testing.T){
    cfg, err := conf.Load("D:\\code\\gosrc\\src\\stash.weimob.com\\devops\\prometheuswriter - 副本\\config.json")
    if err != nil {
        log.Fatal(err)
    }

    strChan := make(chan []byte)
    go PromRemoteWrite(context.Background(),cfg, strChan)

    var timeSeriesList [][]lib.TimeSeries
    timeSeriesList = append(timeSeriesList,[]lib.TimeSeries{
        {
            Labels: []lib.Label{
                {
                    Name:  "__name__",
                    Value: "test_metrics_bucket",
                },
                {
                    Name:  "le",
                    Value: "1000",
                },
                {
                    Name:  "namespace",
                    Value: "test",
                },
            },
            Datapoint: lib.Datapoint{
                Timestamp:  time.Now(),
                Value:     float64(10),
            },
        },
        {
            Labels: []lib.Label{
                {
                    Name:  "__name__",
                    Value: "test_metrics_bucket",
                },
                {
                    Name:  "le",
                    Value: "10000",
                },
                {
                    Name:  "namespace",
                    Value: "test",
                },
            },
            Datapoint: lib.Datapoint{
                Timestamp:  time.Now(),
                Value:     float64(20),
            },
        },
        {
            Labels: []lib.Label{
                {
                    Name:  "__name__",
                    Value: "test_metrics_bucket",
                },
                {
                    Name:  "le",
                    Value: "100000",
                },
                {
                    Name:  "namespace",
                    Value: "test",
                },
            },
            Datapoint: lib.Datapoint{
                Timestamp:  time.Now(),
                Value:     float64(30),
            },
        },
        {
            Labels: []lib.Label{
                {
                    Name:  "__name__",
                    Value: "test_metrics_bucket",
                },
                {
                    Name:  "le",
                    Value: "1e+06",
                },
                {
                    Name:  "namespace",
                    Value: "test",
                },
            },
            Datapoint: lib.Datapoint{
                Timestamp:  time.Now(),
                Value:     float64(40),
            },
        },
        {
            Labels: []lib.Label{
                {
                    Name:  "__name__",
                    Value: "test_metrics_bucket",
                },
                {
                    Name:  "le",
                    Value: "+Inf",
                },
                {
                    Name:  "namespace",
                    Value: "test",
                },
            },
            Datapoint: lib.Datapoint{
                Timestamp:  time.Now(),
                Value:     float64(60),
            },
        },
        {
            Labels: []lib.Label{
                {
                    Name:  "__name__",
                    Value: "test_metrics_sum",
                },
                {
                    Name:  "namespace",
                    Value: "test",
                },
            },
            Datapoint: lib.Datapoint{
                Timestamp:  time.Now(),
                Value:     float64(10410),
            },
        },
        {
            Labels: []lib.Label{
                {
                    Name:  "__name__",
                    Value: "test_metrics_count",
                },
                {
                    Name:  "namespace",
                    Value: "test",
                },
            },
            Datapoint: lib.Datapoint{
                Timestamp:  time.Now(),
                Value:     float64(60),
            },
        },

    })

    for _,v := range timeSeriesList{
        wr := lib.ToPromWriteRequest(v)
        data,err := proto.Marshal(wr)
        if nil != err{
            fmt.Println(err)
        }
        strChan <- data
    }

    time.Sleep(time.Second * 3)

}