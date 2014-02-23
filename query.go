package main

import "net/http"
import "time"

import "fmt"
import "flag"
import (
	"code.google.com/p/plotinum/plot"
	"code.google.com/p/plotinum/plotter"
	"code.google.com/p/plotinum/plotutil"
	"io/ioutil"
	"strconv"
	"sync"
//	"sync/atomic"
)

const (
	queryuri     = "http://localhost"
	queryurins   = "http://localhost:9000/pools/default"
	queryuriview = "http://localhost:9500/default/_design/dev_1/_view/1?stale=false&connection_timeout=60000&limit=10&skip=0"
	MAXWORKER    = 5
	QN           = 40
)

type stats struct {
	maxTime   int64
	minTime   int64
	avgTime   int64
	failures  int
	totalTime int64
}

var status bool
var arr []int64

func sendRequest(numReq int64, st *stats, ch chan bool, cd *sync.Cond) {
	cd.L.Lock()
	for status == false {
		cd.Wait()
	}
	resTime := []int64{}
	totReq := numReq
	for numReq > 0 {
		start := time.Now()
		resp, err := http.Get(queryuri)
		end := int64(time.Since(start))
		resTime = append(resTime, end)
		if err != nil {
			fmt.Println("Error is err", err, "response is ", resp, "restime is", resTime)
			st.failures++
		}
		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		numReq--
	}
	//time.Sleep(1*time.Second)
	var tot, max, min int64
	for _, val := range resTime {
		tot += val
		if val > max {
			max = val
		}
		if val < min {
			min = val
		}
	}
	st.totalTime = tot
	st.avgTime = tot / totReq
	st.maxTime = max
	st.minTime = min
	cd.L.Unlock()
	ch <- true
}

func mainLoop(maxworker int, queryNum int) float64 {
	arr = make([]int64, 10)
	s := make([]stats, maxworker)
	ch := make(chan bool)
	var global_mutex sync.Mutex
	cd := sync.NewCond(&global_mutex)
	status = false
	for i := 0; i < maxworker; i++ {
		go sendRequest(int64(queryNum), &s[i], ch, cd)
	}
	time.Sleep(1 * time.Second)
	status = true
	cd.Broadcast()
	for i := 0; i < maxworker; i++ {
		<-ch
	}
	var maxTime int64
	st := stats{}
	for i := 0; i < maxworker; i++ {
		st.totalTime += s[i].totalTime
		st.failures += s[i].failures
		st.avgTime += s[i].avgTime
		if maxTime < s[i].totalTime {
			maxTime = s[i].totalTime
		}
		if st.maxTime < s[i].maxTime {
			st.maxTime = s[i].maxTime
		}
		if st.minTime > s[i].minTime {
			st.minTime = s[i].minTime
		}
	}
	//AvgTime := float64(st.avgTime/int64(maxworker))/float64(time.Millisecond)
	var totalQuery float64 = float64(queryNum * int(maxworker))
	//qps := (totalQuery / ((float64(st.totalTime) / float64(time.Second))/float64(maxworker)))
	qps := totalQuery / (float64(maxTime) / float64(time.Second))
	//   fmt.Println("Avg Time (mili second)", AvgTime)
	//    fmt.Println("Max Time (mili second)", float64(st.maxTime)/float64(time.Millisecond))
	//    fmt.Println("Min Time (mili second)", st.minTime)
	fmt.Println("worker ", maxworker, " query", queryNum, "QPS", qps, "error", st.failures)
	//cd.L.Unlock()
	return qps
}

func main() {
	var maxworker = *flag.Int("maxworkers", MAXWORKER, "Maxworker")
	var queryNum = *flag.Int("qpr", QN, "query per worker")
	flag.Parse()
	fmt.Println(maxworker, queryNum)
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.Title.Text = "Query per second"
	p.X.Label.Text = "query number"
	p.Y.Label.Text = "query per second"
	pts := make(plotter.XYs, queryNum)
	for i := 10; i <= maxworker; i = i + 3 {
		for j := 1; j <= queryNum; j++ {
			val := mainLoop(i, j)
			pts[j-1].X = float64(j)
			pts[j-1].Y = val
		}
		err = plotutil.AddLinePointsColor(p, i, "Number of worker "+strconv.Itoa(i), pts)
		if err != nil {
			panic(err)
		}
	}
	if err := p.Save(4, 4, "points.png"); err != nil {
		panic(err)
	}
}
