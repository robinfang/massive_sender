package main

import (
    "bufio"
    "bytes"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/http/httptrace"
    "os"
    "strings"
    "time"
)

type Body struct {
    id          string
    transaction string
}
type Result struct {
    id        string
    start     int64
    resp_time int64
}

func check(e error) {
    if e != nil {
        panic(e)
    }
}
func main() {
    content, err := ioutil.ReadFile("../payload_list.txt")
    check(err)
    raw_lines := strings.Split(strings.Replace(string(content), "\r\n", "\n", -1), "\n")
    lines := make([]Body, len(raw_lines))
    for i, el := range raw_lines {
        lines[i] = Body{strings.Split(el, ",")[0], strings.Split(el, ",")[1]}
    }
    fmt.Println(len(lines))
    result_chan := make(chan Result, len(raw_lines)+23)
    body_chan := make(chan Body, len(raw_lines)+23)

    for i := 0; i < 56; i++ {
        go makeRequest(body_chan, result_chan)
    }
    for _, body := range lines {
        body_chan <- body
    }
    close(body_chan)

    f, err := os.Create("../go_request_result.txt")
    check(err)
    defer f.Close()
    w := bufio.NewWriter(f)
    for i := 0; i < len(raw_lines); i++ {
        r := <-result_chan
        // fmt.Println(n)
        // fmt.Printf("%s,%d,%d\n", r.id, r.start, r.resp_time)
        fmt.Fprintf(w, "%s,%d,%d\n", r.id, r.start, r.resp_time)
    }
    w.Flush()

}

func makeRequest(body_chan <-chan Body, result_chan chan<- Result) {
    url := "http://127.0.0.1:8081/transaction/postTranByString"
    for body := range body_chan {
        // fmt.Printf("%v,%v\n", body.id, body.transaction)
        buf := bytes.NewBuffer([]byte(body.transaction))
        req, _ := http.NewRequest("POST", url, buf)
        req.Header.Set("Content-Type", "application/json")
        var start time.Time
        var resp_time time.Duration
        trace := &httptrace.ClientTrace{
            GotFirstResponseByte: func() {
                resp_time = time.Since(start)
            },
        }
        req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
        start = time.Now()
        resp, err := http.DefaultTransport.RoundTrip(req)
        if err != nil {
            log.Fatal(err)
        }
        resp.Body.Close()
        // fmt.Printf("Total time: %v\n", time.Since(start))
        result_chan <- Result{body.id, start.UnixNano(), resp_time.Nanoseconds()}
    }
}
