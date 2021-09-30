package wal

import (
    "context"
    "encoding/binary"
    "io"
    "log"
    "os"
    "path/filepath"
    "sort"
    "sync"
    "time"
)

type pageWriter struct {
    walDir string
    files []string
    bwLock sync.Mutex
    bw *os.File
    curFileIndex uint64
    curOff int64
    curItem uint64
    maxItem uint64

    buf []byte
    bufferedBytes int
    uint64buf []byte
}

func newPageWriter(walDir string, maxItem uint64)*pageWriter{
    pw := &pageWriter{}
    var err error
    var f *os.File
    var newFile string
    var seq uint64

    if _,err := os.Stat(walDir);nil != err{
        if os.IsNotExist(err){
            if err := os.Mkdir(walDir,0666);nil != err{
                log.Fatalf("mkdir %s err:%s",walDir,err.Error())
            }
        }else {
            log.Fatalf("walDir %s stat err:%s",walDir,err.Error())
        }
    }

    files,err := readDir(walDir)
    if nil != err{
        log.Fatalf("read waldir %s err:%s",walDir,err.Error())
    }

    lenFiles := len(files)
    if 0 != lenFiles{
        seq,_,err = parseWALName(files[lenFiles-1])
        if nil != err {
            log.Fatalf("parse wal name %s err %s",files[lenFiles-1],err.Error())
        }
    }else {
        seq = 0
    }

    newFile = filepath.Join(walDir, walName(seq + 1, 0))
    if _, err := os.Stat(newFile);os.ErrExist == err{
        os.Remove(newFile)
    }

    f,err = os.Create(newFile)
    if nil != err{
        log.Fatalf("create new wal file %s err %s",newFile,err.Error())
    }

    curOff,err := f.Seek(0, io.SeekCurrent)
    if nil != err{
        f.Close()
        log.Fatalf(err.Error())
    }

    pw.walDir = walDir
    pw.files = append(files, newFile)
    pw.bw = f
    pw.curFileIndex = seq + 1
    pw.curOff = curOff
    pw.curItem = 0
    pw.maxItem = maxItem

    pw.buf = make([]byte,pageBytes)
    pw.bufferedBytes = 0
    pw.uint64buf = make([]byte,8)

    return pw
}

func (pw *pageWriter)flush(data []byte) (n int, err error){
    lenField, padBytes := encodeFrameSize(len(data))
    if padBytes != 0 {
        data = append(data, make([]byte, padBytes)...)
    }

    writeUint64(lenField, pw.uint64buf)

    pw.bwLock.Lock()
    if pw.bufferedBytes + 8 + len(data) <= len(pw.buf){
        copy(pw.buf[pw.bufferedBytes:], pw.uint64buf)
        copy(pw.buf[pw.bufferedBytes+8:], data)
        pw.bufferedBytes += 8 + len(data)
        pw.bwLock.Unlock()
        pw.curItem ++
        return
    }

    // pw.buf已满，需要将buf写入磁盘
    if pw.curOff + int64(len(pw.buf)) < segmentSizeBytes{
        // 此处的write可能发生在runFlush中的 close 操作之后，此时会尝试写入已关闭的文件。由于在close中已经
        // 将数据刷新到磁盘，因此此时忽略write错误即可
        pw.write()
        pw.flush(data)
        return
    }

    // 当前文件已满，关闭并创建新文件
    // 注意close的同时，有可能会在runFlush中发生write操作，此时会尝试写入已关闭的文件。由于在close中已经
    // 将数据刷新到磁盘，因此此时忽略write错误即可
    pw.close()

    p := filepath.Join(pw.walDir, walName(pw.curFileIndex + 1 , 0))
    f,err := os.Create(filepath.Join(pw.walDir, walName(pw.curFileIndex + 1 , 0)))
    if nil != err {
        return -1,err
    }

    err = f.Truncate(segmentSizeBytes)
    if nil != err{
        f.Close()
        os.Remove(p)
        return -1, err
    }

    pw.files = append(pw.files, p)
    pw.curFileIndex ++

    pw.bwLock.Lock()
    pw.bw = f
    pw.bwLock.Unlock()

    return pw.flush(data)
}

func (pw *pageWriter)rmOutRangeWal(maxItem uint64) {
    var seqSlice []int

    seqMap := make(map[uint64]string)
    fileMap := make(map[string]uint64)

    for _,fName := range pw.files{
        seq,itemNum,err := parseWALName(fName)
        if nil != err{
            log.Printf("parse file %s err %s",fName,err.Error())
            continue
        }

        seqSlice = append(seqSlice, int(seq))
        seqMap[seq] = fName
        fileMap[fName] = itemNum
    }

    sort.Ints(seqSlice)

    var totalItem uint64 = 0
    var rmEndIndex int
    lenSeqSlice := len(seqSlice)
    for i:=lenSeqSlice-1; i>=0; i--{
        seq := uint64(seqSlice[i])
        totalItem +=  fileMap[seqMap[seq]]
        if totalItem > maxItem{
            rmEndIndex = i
            break
        }
    }

    for _,v := range pw.files[:rmEndIndex]{
        os.Remove(v)
    }

    pw.files = pw.files[rmEndIndex:]
    return
}

func (pw *pageWriter)write() (int, error){
    pw.bwLock.Lock()
    defer pw.bwLock.Unlock()

    if 0 == pw.bufferedBytes{
        return 0,nil
    }

    n,err := pw.bw.Write(pw.buf[:pw.bufferedBytes])
    if nil != err{
        return -1,err
    }

    pw.updateStat(int64(len(pw.buf[:pw.bufferedBytes])))
    return n,nil
}

func (pw *pageWriter)updateStat(dataLen int64){
    pw.curOff += dataLen

    pw.bufferedBytes = 0
    binary.LittleEndian.PutUint64(pw.uint64buf,0)
}

func (pw *pageWriter)clearStat(){
    pw.curItem = 0
    pw.curOff = 0

    pw.bufferedBytes = 0
    binary.LittleEndian.PutUint64(pw.uint64buf,0)
}

func (pw *pageWriter)close(){
    pw.bwLock.Lock()
    defer pw.bwLock.Unlock()

    pw.bw.Write(pw.buf[:pw.bufferedBytes])

    off, err :=  pw.bw.Seek(0, io.SeekCurrent)
    if err != nil {
        log.Printf("seek current offset for file %s err:%s",pw.bw.Name())
    }else{
        pw.bw.Truncate(off)
    }

    pw.bw.Close()

    if 0 == pw.curItem{
        os.Remove(filepath.Join(pw.walDir, walName(pw.curFileIndex , 0)))
    }else{
        err = pw.rename(filepath.Join(pw.walDir, walName(pw.curFileIndex , 0)), filepath.Join(pw.walDir, walName(pw.curFileIndex , pw.curItem)))
        if nil != err{
            log.Printf("rename file %s to %s err:%s",walName(pw.curFileIndex , 0),walName(pw.curFileIndex , pw.curItem),err)
        }
    }

    pw.clearStat()
    pw.rmOutRangeWal(pw.maxItem)
}

func (pw *pageWriter)rename(oldName,newName string)error{
    err := os.Rename(oldName, newName)
    if nil != err{
        return err
    }
    for k,v := range pw.files{
        if v == oldName{
            pw.files[k] = newName
        }
    }

    return nil
}

type encoder struct {
    writeTicker *time.Ticker

    pw *pageWriter
}

func (e *encoder)runFlush(ctx context.Context){
    go func() {
        for {
            select {
            case <- e.writeTicker.C:
                e.pw.write()
            case <- ctx.Done():
                e.pw.close()
                e.writeTicker.Stop()
                return
            }
        }
    }()
}

func (e *encoder)Write(data []byte)(n int, err error){
    return e.pw.flush(data)
}