package lib

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/prometheus/pkg/exemplar"
    "github.com/prometheus/prometheus/pkg/labels"
    "github.com/prometheus/prometheus/tsdb"
    conf "github.com/woodliu/prometheusWriter/pkg/config"
    "github.com/woodliu/prometheusWriter/pkg/prompb"
    "log"
)

var eMetrics = tsdb.NewExemplarMetrics(prometheus.DefaultRegisterer)

func NewCircularExemplarStorage(cfg *conf.Conf)*tsdb.CircularExemplarStorage{
    exs, err := tsdb.NewCircularExemplarStorage(int64(cfg.Exemplars.MaxExemplars), eMetrics)
    if nil != err{
        log.Fatal(err.Error())
    }

    return exs.(*tsdb.CircularExemplarStorage)
}

func AddNewExemplar(es *tsdb.CircularExemplarStorage, pbLabels []prompb.Label, e []prompb.Exemplar){
    getLabels := func(pbLs []prompb.Label) labels.Labels{
        var ls labels.Labels
        for _,v := range pbLs{
            ls = append(ls,labels.Label{
                Name: v.Name,
                Value: v.Value,
            })
        }
        return ls
    }

    for _,v := range e{
        exem := exemplar.Exemplar{
            Labels: getLabels(v.Labels),
            Value: v.Value,
            Ts: v.Timestamp,
        }

        if 0 != v.Timestamp{
            exem.HasTs = true
        }else{
            exem.HasTs = false
        }

        if err := es.AddExemplar(getLabels(pbLabels), exem);nil != err{
            log.Println(err)
        }
    }

    return
}

func SelectExemplar(es *tsdb.CircularExemplarStorage, start, end int64, ls labels.Labels)([]exemplar.QueryResult, error){
    var m []*labels.Matcher
    for _,l := range ls{
        m = append(m ,labels.MustNewMatcher(labels.MatchEqual, l.Name, l.Value))
    }

    return es.Select(start,end,m)
}