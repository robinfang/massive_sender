package main

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/http/httptrace"
    "strings"
    "time"
)

type Body struct {
    id          string
    transaction string
}

func main() {

    content, err := ioutil.ReadFile("file.txt")
    if err != nil {
        //Do something
    }
    raw_lines := strings.Split(strings.Replace(string(content), "\r\n", "\n", -1), "\n")
    lines := make([]Body, len(raw_lines))
    for i, el := range raw_lines {
        lines[i] = Body{strings.Split(el, ",")[0], strings.Split(el, ",")[1]}
    }
    results := make(chan float64)
    //TODO 需要对并发数量加以限制
    for _, el := range lines {
        go func(el Body) {
            results <- makeRequest(el)
        }(el)
    }
    for i := 0; i < len(lines); i++ {
        fmt.Printf("%v\n", <-results)
    }

}
func makeRequest(body Body) float64 {
    url := "http://localhost:8081/transaction/postTranByString"
    fmt.Printf("%v,%v\n", body.id, body.transaction)
    buf := bytes.NewBuffer([]byte(body.transaction))
    req, _ := http.NewRequest("POST", url, buf)
    req.Header.Set("Content-Type", "application/json")
    var start time.Time
    var resp_time time.Duration
    trace := &httptrace.ClientTrace{
        GotFirstResponseByte: func() {
            resp_time = time.Since(start)
            fmt.Printf("Time from start to first byte: %v\n", resp_time)
        },
    }
    req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
    start = time.Now()
    if _, err := http.DefaultTransport.RoundTrip(req); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Total time: %v\n", time.Since(start))
    return resp_time.Seconds()

}
