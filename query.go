package main

import "net/http"
import "time"

import "fmt"
import "flag"
import (
	"code.google.com/p/plotinum/plot"
	"code.google.com/p/plotinum/plotter"
	"code.google.com/p/plotinum/plotutil"
    "strconv"
)

const (
	queryuri  = "http://localhost:9500/default/_design/dev_1/_view/1?stale=false&connection_timeout=60000&limit=10&skip=0"
	MAXWORKER = 3
	QN        = 40
)

type stats struct {
	maxTime   int64
	minTime   int64
	avgTime   int64
	failures  int
	totalTime int64
}

func sendRequest(numReq int64, st *stats, ch chan bool) {
	resTime := []int64{}
	totReq := numReq
	for numReq > 0 {
		start := time.Now()
	    resp, err := http.Get(queryuri)
		end := int64(time.Since(start))
		resTime = append(resTime, end)
		if err != nil {
            fmt.Println("Error is err", err, "response is ", resp)
			st.failures++
		}
		numReq--
	}
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
	ch <- true
}

func mainLoop(maxworker int, queryNum int) float64 {
	s := make([]stats, maxworker)
	ch := make(chan bool)
	for i := 0; i < maxworker; i++ {
		go sendRequest(int64(queryNum), &s[i], ch)
	}
	for i := 0; i < maxworker; i++ {
		<-ch
	}
	st := stats{}
	for i := 0; i < maxworker; i++ {
		st.totalTime += s[i].totalTime
		st.failures += s[i].failures
		st.avgTime += s[i].avgTime
		if st.maxTime < s[i].maxTime {
			st.maxTime = s[i].maxTime
		}
		if st.minTime > s[i].minTime {
			st.minTime = s[i].minTime
		}
	}
	//AvgTime := float64(st.avgTime/int64(maxworker))/float64(time.Millisecond)
	var totalQuery float64 = float64(queryNum * int(maxworker))
	qps := (totalQuery / (float64(st.totalTime) / float64(time.Second)))
	//   fmt.Println("Avg Time (mili second)", AvgTime)
	//    fmt.Println("Max Time (mili second)", float64(st.maxTime)/float64(time.Millisecond))
	//    fmt.Println("Min Time (mili second)", st.minTime)
	fmt.Println("worker ",maxworker," query", queryNum, "QPS",qps, "error", st.failures)
	return qps
}

func main() {
	var maxworker = *flag.Int("maxworkers", MAXWORKER, "Maxworker")
	var queryNum = *flag.Int("qpr", QN, "query per worker")
	flag.Parse()
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.Title.Text = "Query per second"
	p.X.Label.Text = "query number"
	p.Y.Label.Text = "query per second"
	pts := make(plotter.XYs, queryNum)
	for i := 1; i <= maxworker; i++ {
		for j := 1; j <= queryNum; j++ {
			val := mainLoop(i, j)
			pts[i].X = float64(j)
			pts[i].Y = val
		}
		err = plotutil.AddLinePoints(p, "Number of worker "+strconv.Itoa(i), pts)
		if err != nil {
			panic(err)
		}
	}
	if err := p.Save(4, 4, "points.png"); err != nil {
		panic(err)
	}
}
