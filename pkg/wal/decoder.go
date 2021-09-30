package wal

import (
    "io"
    "log"
    "os"
)

type chanResp struct {
    req chan []byte
    resp chan struct{}
}

type Decoder struct {
    buf []byte
    files []string
    loadChan chanResp
}

func (d *Decoder)LoadWal(){
    go func() {
        for _,v := range d.files{
            f,err := os.Open(v)
            if nil != err{
                log.Printf("read file %s,err:%s",f,err.Error())
                continue
            }

            for{
                frame,err := readInt64(f)
                if nil != err {
                    if io.EOF == err{
                        goto NextFile
                    }

                    log.Printf("read file %s frame info,err:%s",f,err.Error())
                    goto NextFile
                }

                dataLen,padsLen := decodeFrameSize(frame)
                if 0 == dataLen{
                    goto NextFile
                }

                var buf []byte
                if dataLen + padsLen <= int64(len(d.buf)){
                    buf = d.buf
                } else{
                    buf = make([]byte,dataLen + padsLen)
                }
                _,err = io.ReadFull(f, buf[:(dataLen+padsLen)])
                if nil != err{
                    log.Printf("decode frame file %s frame info,err:%s",f,err.Error())
                    goto NextFile
                }

                d.loadChan.req <- d.buf[:dataLen]
                <- d.loadChan.resp
            }
        NextFile:
            f.Close()
        }
        close(d.loadChan.req)
    }()
}