package wal

import (
    "context"
    "github.com/gogo/protobuf/proto"
    "github.com/prometheus/prometheus/tsdb"
    "github.com/woodliu/prometheusWriter/pkg/lib"
    "github.com/woodliu/prometheusWriter/pkg/prompb"
    "log"
    "time"
)

var (
    WalDir = "/wal"
    flushInterval = time.Second * 15
    segmentSizeBytes int64 = 64 * 1024 * 1024 // 64MB
    pageBytes = 4 * 1024  //4KB
)

func NewDecoder(walDir string)(*Decoder,error){
    var decoder Decoder
    var err error

    decoder.loadChan = chanResp{
        req:make(chan []byte),
        resp: make(chan struct{}),
    }
    decoder.buf = make([]byte, pageBytes)
    decoder.files,err = readDir(walDir)
    return &decoder,err
}

func NewEncoder(ctx context.Context, walDir string, maxItem uint64)*encoder{
    var e encoder

    e.pw = newPageWriter(walDir, maxItem)
    e.writeTicker = time.NewTicker(flushInterval)
    e.runFlush(ctx)
    return &e
}

func Replay(es *tsdb.CircularExemplarStorage){
    var wr prompb.WriteRequest

    d,err := NewDecoder(WalDir);
    if nil != err {
        log.Println(err)
        return
    }

    d.LoadWal()
    for data := range d.loadChan.req{
        err := proto.Unmarshal(data, &wr)
        if nil != err{
            log.Println(err)
            continue
        }
        for _,v := range wr.Timeseries{
            lib.AddNewExemplar(es, v.Labels, v.Exemplars)
        }
        d.loadChan.resp <- struct{}{}
    }
}