package main

import (
    "bufio"
    "bytes"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "strings"
    "sync"
    "time"
)

type bodyType struct {
    id          string
    transaction string
}
type resultType struct {
    id       string
    start    int64
    respTime int64
}

func check(e error) {
    if e != nil {
        log.Fatal(e)
    }
}

var httpClient *http.Client

func init() {
    httpClient = &http.Client{
        Timeout: 10 * time.Second,
    }
}

func main() {

    var wg sync.WaitGroup
    content, err := ioutil.ReadFile("../payload_list.txt")
    check(err)
    rawLines := strings.Split(strings.Replace(string(content), "\r\n", "\n", -1), "\n")
    fmt.Println(len(rawLines))
    lines := make([]bodyType, 0)
    // fmt.Printf("len lines %v, cap=%d \n", len(lines), cap(lines))
    for _, el := range rawLines {
        if len(el) > 0 {
            lines = append(lines, bodyType{strings.Split(el, ",")[0], strings.Split(el, ",")[1]})
        }
    }
    fmt.Printf("len lines %v, cap=%d \n", len(lines), cap(lines))
    resultChan := make(chan resultType, len(lines)+23)
    bodyChan := make(chan bodyType, len(lines)+23)
    fmt.Println("go makeAllRequests")
    for i := 0; i < 48; i++ {
        wg.Add(1)
        go makeAllRequests(bodyChan, resultChan, &wg)
    }
    fmt.Println("pushing into bodyChan")
    for _, body := range lines {
        bodyChan <- body
    }
    close(bodyChan)
    // go func() {
    //     for {
    //         time.Sleep(1 * time.Second)
    //         fmt.Printf("len bodychan: %v, len resultchan: %v\n", len(bodyChan), len(resultChan))
    //     }

    // }()
    fmt.Println("waiting")
    wg.Wait()
    fmt.Println("close resultChan")
    close(resultChan)
    f, err := os.Create("../go_request_result.txt")
    check(err)
    defer f.Close()
    w := bufio.NewWriter(f)
    n := 0
    for r := range resultChan {
        n++
        if n%100 == 0 {
            fmt.Println(n)
        }
        // fmt.Printf("%s,%d,%d\n", r.id, r.start, r.respTime)
        fmt.Fprintf(w, "%s,%d,%d\n", r.id, r.start, r.respTime)
    }
    w.Flush()

}

func makeOne(url string, body bodyType, resultChan chan<- resultType) {
    buf := bytes.NewBuffer([]byte(body.transaction))
    start := time.Now()
    resp, err := httpClient.Post(url, "application/json", buf)
    if err != nil {
        log.Println(err)

    }
    ioutil.ReadAll(resp.Body)

    defer resp.Body.Close()
    respTime := time.Since(start)

    // fmt.Printf("Total time: %v\n", time.Since(start))
    resultChan <- resultType{body.id, start.UnixNano(), respTime.Nanoseconds()}

}

func makeAllRequests(bodyChan <-chan bodyType, resultChan chan<- resultType, wg *sync.WaitGroup) {
    defer wg.Done()
    url := "http://192.168.2.70:8081/transaction/postTranByString"
    for body := range bodyChan {
        // fmt.Printf("%v,%v\n", body.id, body.transaction)
        makeOne(url, body, resultChan)
    }
}
