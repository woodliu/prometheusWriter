package app

import (
    "encoding/json"
    "github.com/pkg/errors"
    "github.com/prometheus/prometheus/pkg/timestamp"
    "github.com/prometheus/prometheus/promql/parser"
    "github.com/prometheus/prometheus/tsdb"
    conf "github.com/woodliu/prometheusWriter/pkg/config"
    "io/ioutil"
    "log"
    "math"
    "net/http"
    "strconv"
    "time"
)

type QueryExemplar struct{
    Start string `json:"start"`
    End   string `json:"end"`
    Query string `json:"query"`
}

type server struct {
    es *tsdb.CircularExemplarStorage
}

func NewExemplarSrv(cfg *conf.Conf,es *tsdb.CircularExemplarStorage)*http.Server{
    return &http.Server{
        Addr: ":" + strconv.Itoa(cfg.Exemplars.Port),
        Handler: &server{
            es: es,
        },
    }
}

func (h *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    body,err := ioutil.ReadAll(r.Body)
    if nil != err {
        log.Println(err)
        goto END
    }
    defer r.Body.Close()

    {
        var qe QueryExemplar
        if err := json.Unmarshal(body, &qe);nil != err{
            log.Printf("json unmarshal request err:%s",err.Error())
            goto END
        }

        start, err := parseTime(qe.Start)
        if err != nil {
            log.Printf("Invalid time value for '%s',err:%s", qe.Start,err)
            goto END
        }

        end, err := parseTime(qe.End)
        if err != nil {
            log.Printf("Invalid time value for '%s',err:%s", qe.End,err)
            goto END
        }

        if end.Before(start){
            log.Printf("end time before start time")
            goto END
        }

        expr, err := parser.ParseExpr(qe.Query)
        if err != nil {
            log.Printf("query '%s' err:%s",qe.Query, err)
            goto END
        }

        selectors := parser.ExtractSelectors(expr)
        if len(selectors) < 1 {
            log.Printf("selector num < 1")
            goto END
        }

        ret, err := h.es.Select(timestamp.FromTime(start)/1000, timestamp.FromTime(end)/1000, selectors...)
        if err != nil {
            log.Printf("query exemplar err:%s",err)
            return
        }

        resp,_ := json.Marshal(ret)
        w.WriteHeader(http.StatusOK)
        w.Write(resp)
        return
    }

END:
    w.WriteHeader(http.StatusBadRequest)
    return
}

var (
    minTime = time.Unix(math.MinInt64/1000+62135596801, 0).UTC()
    maxTime = time.Unix(math.MaxInt64/1000-62135596801, 999999999).UTC()

    minTimeFormatted = minTime.Format(time.RFC3339Nano)
    maxTimeFormatted = maxTime.Format(time.RFC3339Nano)
)

func parseTime(s string) (time.Time, error) {
    if t, err := strconv.ParseFloat(s, 64); err == nil {
        s, ns := math.Modf(t)
        ns = math.Round(ns*1000) / 1000
        return time.Unix(int64(s), int64(ns*float64(time.Second))).UTC(), nil
    }
    if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
        return t, nil
    }

    // Stdlib's time parser can only handle 4 digit years. As a workaround until
    // that is fixed we want to at least support our own boundary times.
    // Context: https://github.com/prometheus/client_golang/issues/614
    // Upstream issue: https://github.com/golang/go/issues/20555
    switch s {
    case minTimeFormatted:
        return minTime, nil
    case maxTimeFormatted:
        return maxTime, nil
    }
    return time.Time{}, errors.Errorf("cannot parse %q to a valid timestamp", s)
}