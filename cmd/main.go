package main

import (
    "context"
    "github.com/prometheus/prometheus/tsdb"
    "github.com/woodliu/prometheusWriter/pkg/lib"
    "github.com/woodliu/prometheusWriter/pkg/wal"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/woodliu/prometheusWriter/pkg/app"
    conf "github.com/woodliu/prometheusWriter/pkg/config"
    "log"
)

func main() {
    cfg, err := conf.Load("./config.json")
    if err != nil {
        log.Fatal(err)
    }

    var es *tsdb.CircularExemplarStorage
    if cfg.Exemplars.Enable{
        es = lib.NewCircularExemplarStorage(cfg)
    }

    wal.Replay(es)

    ctx,cancel := context.WithCancel(context.Background())
    strChan := make(chan []byte)
    go app.KfaConsumer(ctx, cfg, strChan, es)
    go app.PromRemoteWrite(ctx, cfg, strChan)

    srv := app.NewExemplarSrv(cfg,es)
    go func() {
        log.Println("Prometheus Writer starting!")
        if err := srv.ListenAndServe(); err != nil {
            log.Fatal(err)
        }
    }()

    sigNotify(cancel,srv)
}

func sigNotify(cancel context.CancelFunc,srv *http.Server){
    c := make(chan os.Signal)
    signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
    s := <-c

    log.Printf("system shutdown by signal:[%s]",s.String())
    cancel()
    time.Sleep(time.Second * 3) //等待buf刷新到磁盘
    srv.Shutdown(context.Background())
}