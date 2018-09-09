package gbench

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Gbench struct {
	url                 string
	concurrency         uint64
	requests            uint64
	startTime           time.Time
	count 				*Count
}

type Count struct {
	successCount        uint64
	errorCount          uint64
	totalRequestUseTime uint64
	totalRecvData       uint64
}

func New(url string, concurrency uint64, requests uint64) *Gbench {
	if len(url) == 0 {
		fmt.Println("url not empty")
		os.Exit(1)
	}
	if requests <= 0 {
		fmt.Println("requests number must be greater than 0")
		os.Exit(1)
	}
	if requests > 0 && requests < concurrency {
		fmt.Println("requests number must be greater than the number of concurrency")
		os.Exit(1)
	}

	r := new(Gbench)
	r.url = url
	r.concurrency = concurrency
	r.requests = requests
	r.count = new(Count)

	return r
}

func (r *Gbench) addSuccessCount() {
	atomic.AddUint64(&r.count.successCount, 1)
}

func (r *Gbench) addErrorCount() {
	atomic.AddUint64(&r.count.errorCount, 1)
}

func (r *Gbench) addRecvData(num uint64) {
	atomic.AddUint64(&r.count.totalRecvData, num)
}

func (r *Gbench) addRequestTime(num uint64) {
	atomic.AddUint64(&r.count.totalRequestUseTime, num)
}

func (r *Gbench) runner(wg *sync.WaitGroup, requests int) {
	defer wg.Done()
	for i := 0; i < requests; i++ {
		start := time.Now()
		client := &http.Client{
			Timeout: 30 * time.Second,
		}
		resp, err := client.Get(r.url)
		if err != nil {
			r.addErrorCount()
			continue
		}
		if resp.StatusCode != 200 {
			r.addErrorCount()
			continue
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			r.addErrorCount()
			continue
		}
		useTime := time.Now().Sub(start)
		r.addRequestTime(uint64(useTime))
		r.addRecvData(uint64(len(data)))
		r.addSuccessCount()
	}
}

func (r *Gbench) Run() {
	r.progress()
	r.signal()

	r.startTime = time.Now()
	tail := r.requests % r.concurrency
	oneTaskRequest := r.requests / r.concurrency

	wg := new(sync.WaitGroup)
	for i := 0; i < int(r.concurrency); i++ {
		wg.Add(1)
		reqs := oneTaskRequest
		if tail != 0 {
			reqs += 1
			tail--
		}

		go r.runner(wg, int(reqs))
	}

	wg.Wait()
	fmt.Print(r.String())
}

func (r *Gbench) progress() {
	go func() {
		for {
			p := int(r.count.successCount+r.count.errorCount) * 100 / int(r.requests)
			fmt.Printf("\rprogress: %3d%%", p)
			time.Sleep(time.Second)
		}
	}()
}

func (r *Gbench) signal() {
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)

		<-c
		fmt.Print(r.String())
		os.Exit(0)
	}()
}

func (r *Gbench) String() string {
	useTime := time.Now().Sub(r.startTime)
	qps := float64(r.count.successCount + r.count.errorCount) / useTime.Seconds()
	tpr := r.count.totalRequestUseTime / uint64(r.count.successCount + r.count.errorCount)

	s := fmt.Sprintf("\r              \n\n")
	s += fmt.Sprintf("Total requests: %v\n", r.requests)
	s += fmt.Sprintf("Complete requests: %v\n", (r.count.successCount + r.count.errorCount))
	s += fmt.Sprintf("Success requests: %v\n", r.count.successCount)
	s += fmt.Sprintf("Error requests: %v\n", r.count.errorCount)
	s += fmt.Sprintf("Time taken for tests: %v\n", useTime)
	s += fmt.Sprintf("Requests per second(mean): %.2f\n", qps)
	s += fmt.Sprintf("Time per request(mean): %v\n", time.Duration(tpr))
	s += fmt.Sprintf("Total recv html body(byte): %v\n", r.count.totalRecvData)

	return s
}
