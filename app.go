package main

import (
    "bufio"
    "bytes"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "strings"
    "time"
    "log"
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
        log.Fatal(e)
    }
}

var httpClient *http.Client

func init() {
    httpClient = &http.Client{
        Timeout: 0 * time.Second,
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

    for i := 0; i < 32; i++ {
        go makeAllRequests(body_chan, result_chan)
    }
    for _, body := range lines {
        body_chan <- body
    }
    close(body_chan)

    f, err := os.Create("../go_request_result.txt")
    check(err)
    defer f.Close()
    w := bufio.NewWriter(f)
    n := 0
    for i := 0; i < len(raw_lines); i++ {
        r := <-result_chan
        n++
        fmt.Println(n)
        // fmt.Printf("%s,%d,%d\n", r.id, r.start, r.resp_time)
        fmt.Fprintf(w, "%s,%d,%d\n", r.id, r.start, r.resp_time)
    }
    w.Flush()

}

func makeOne(url string, body Body, result_chan chan<- Result){
     buf := bytes.NewBuffer([]byte(body.transaction))
        start := time.Now()
        resp, err := httpClient.Post(url, "application/json", buf)
        if err!=nil{
            log.Println(err)
        }
        ioutil.ReadAll(resp.Body)

        defer resp.Body.Close()
        resp_time := time.Since(start)
        
        // fmt.Printf("Total time: %v\n", time.Since(start))
        result_chan <- Result{body.id, start.UnixNano(), resp_time.Nanoseconds()}

}

func makeAllRequests(body_chan <-chan Body, result_chan chan<- Result) {
    url := "http://192.168.2.70:8081/transaction/postTranByString"
    for body := range body_chan {
        // fmt.Printf("%v,%v\n", body.id, body.transaction)
        makeOne(url, body, result_chan)
    }
}
