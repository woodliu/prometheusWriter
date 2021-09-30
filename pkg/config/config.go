package config

import (
    "encoding/json"
    "io/ioutil"
    "os"
)

type Conf struct {
    Kafka struct {
        Auth struct {
            InstanceID string `json:"instanceId"`
            Password   string `json:"password"`
            UserName   string `json:"userName"`
        } `json:"auth"`
        GroupID string   `json:"groupId"`
        Servers []string `json:"servers"`
        Topics  []string `json:"topics"`
    } `json:"kafka"`
    Exemplars struct {
        Port         int   `json:"port"`
        Enable       bool  `json:"enable"`
        MaxExemplars uint64 `json:"maxExemplars"`
    } `json:"exemplars"`
    RemoteURL string `json:"remoteUrl"`
}

func Load(path string)(*Conf,error){
    f,err := os.Open(path)
    if nil != err {
        return nil,err
    }

    defer f.Close()
    c,err := ioutil.ReadAll(f)
    if nil != err {
        return nil,err
    }

    var conf Conf
    err = json.Unmarshal(c,&conf)
    if nil != err {
        return nil,err
    }

    return &conf,nil
}