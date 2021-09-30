package app

import (
    "context"
    "time"

    conf "github.com/woodliu/prometheusWriter/pkg/config"
    "github.com/woodliu/prometheusWriter/pkg/lib"
    "log"
)

var (
    WRITE_TIMEOUT = time.Second * 15
)

func PromRemoteWrite(ctx context.Context,cfg *conf.Conf, msgChan <- chan []byte){
    // create config and client
    promCfg := lib.NewConfig(
        lib.WriteURLOption(cfg.RemoteURL),
        lib.HTTPClientTimeoutOption(WRITE_TIMEOUT),
        lib.UserAgent("metric_collector"),
    )

    client, err := lib.NewClient(promCfg)
    if err != nil {
        log.Fatalf("unable to construct client: %v", err)
    }

    for{
        select {
        case msgData := <- msgChan:
            _,err := client.WriteRaw(context.Background(),msgData,lib.WriteOptions{})
            if nil != err{
                log.Printf("remote write err:%s, errCode:%d",err.Error(),err.StatusCode())
            }
        case <- ctx.Done():
            return
        }
    }
}
