package main

import (
    "bufio"
    "bytes"
    "flag"
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
        log.Println(e)
    }
}

var httpClient *http.Client

func init() {
    httpClient = &http.Client{
        Timeout: 60 * time.Second,
    }
}

func main() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)
    fmt.Println("务必检查服务器时间与本机时间是否同步！最好利用ntp服务进行同步。")
    var hostAddrP = flag.String("h", "", "hostAddr:port")
    var nOfW = flag.Int("n", 1, "number of workers")
    var interval = flag.Int("i", 0, "interval in int /time.Millisecond (optional)")
    flag.Parse()
    fmt.Printf("hostAddr: %s\n", *hostAddrP) // For debug
    fmt.Printf("number of workers: %d\n", *nOfW)
    fmt.Printf("interval: %d\n", *interval)
    if len(*hostAddrP) == 0 {
        fmt.Println("hostAddr:port must be provided")
        os.Exit(0)
    }
    var wg sync.WaitGroup
    content, err := ioutil.ReadFile("../payload_list.txt")
    check(err)
    rawLines := strings.Split(strings.Replace(string(content), "\r\n", "\n", -1), "\n")
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
    for i := 0; i < *nOfW; i++ {
        wg.Add(1)
        go makeAllRequests(hostAddrP, interval, bodyChan, resultChan, &wg)
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
    // n := 0
    fmt.Println("writing to file")
    for r := range resultChan {
        // n++
        // if n%100 == 0 {
        //     fmt.Println(n)
        // }
        // fmt.Printf("%s,%d,%d\n", r.id, r.start, r.respTime)
        fmt.Fprintf(w, "%s,%d,%d\n", r.id, r.start, r.respTime)
    }
    fmt.Println("flushing")
    w.Flush()

}

func makeOne(url string, body bodyType, interval *int, resultChan chan<- resultType) {
    buf := bytes.NewBuffer([]byte(body.transaction))
    start := time.Now()
    resp, err := httpClient.Post(url, "application/json", buf)
    if err != nil {
        log.Println(err)
        return
    }
    _, err = ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Println(err)
    }
    defer resp.Body.Close()
    respTime := time.Since(start)
    // fmt.Printf("Total time: %v\n", time.Since(start))
    resultChan <- resultType{body.id, start.UnixNano(), respTime.Nanoseconds()}

}

func makeAllRequests(hostAddr *string, interval *int, bodyChan <-chan bodyType, resultChan chan<- resultType, wg *sync.WaitGroup) {
    defer wg.Done()
    url := fmt.Sprintf("http://%s/transaction/postTranByString", *hostAddr)
    for body := range bodyChan {
        // fmt.Printf("%v,%v\n", body.id, body.transaction)
        makeOne(url, body, interval, resultChan)
        if *interval > 0 {
            time.Sleep((time.Duration(*interval)) * time.Millisecond)
        }
    }
}
