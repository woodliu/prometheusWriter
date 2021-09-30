package wal

import (
    "encoding/binary"
    "errors"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

func readDir(walDir string)([]string,error){
    dir,err := os.Open(walDir)
    if nil != err{
        return nil,err
    }
    defer dir.Close()

    names,err := dir.Readdirnames(-1)
    if nil != err{
        return nil,err
    }

    sort.Strings(names)

    filterFile := make([]string, 0)
    for _,f := range names{
        if filepath.Ext(f) == ".wal"{
            filterFile = append(filterFile,filepath.Join(walDir, f))
        }
    }

    return filterFile,nil
}

func writeUint64(n uint64, buf []byte) {
    // http://golang.org/src/encoding/binary/binary.go
    binary.LittleEndian.PutUint64(buf, n)
}

func readInt64(r io.Reader) (int64, error) {
    var n int64
    err := binary.Read(r, binary.LittleEndian, &n)
    return n, err
}

func walName(seq, index uint64) string {
    return fmt.Sprintf("%016x-%016x.wal", seq, index)
}

func parseWALName(str string) (seq, index uint64, err error) {
    if !strings.HasSuffix(str, ".wal") {
        return 0, 0, errors.New("")
    }

    _,fName := filepath.Split(str)
    _, err = fmt.Sscanf(fName, "%016x-%016x.wal", &seq, &index)
    return seq, index, err
}

func encodeFrameSize(dataBytes int) (lenField uint64, padBytes int) {
    lenField = uint64(dataBytes)
    // force 8 byte alignment so length never gets a torn write
    padBytes = (8 - (dataBytes % 8)) % 8
    if padBytes != 0 {
        lenField |= uint64(0x80|padBytes) << 56
    }
    return lenField, padBytes
}

func decodeFrameSize(lenField int64) (recBytes int64, padBytes int64) {
    // the record size is stored in the lower 56 bits of the 64-bit length
    recBytes = int64(uint64(lenField) & ^(uint64(0xff) << 56))
    // non-zero padding is indicated by set MSb / a negative length
    if lenField < 0 {
        // padding is stored in lower 3 bits of length MSB
        padBytes = int64((uint64(lenField) >> 56) & 0x7)
    }
    return recBytes, padBytes
}
