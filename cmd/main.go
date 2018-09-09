package main

import (
	"flag"
	"github.com/sinchie/gbench"
)

// 并发数
var concurrent = flag.Uint64("c", 1, "Number of multiple requests to make")
// 总请求量
var requests = flag.Uint64("n", 0, "Number of requests to perform")
// 请求地址
var url = flag.String("u", "", "Request url")

func main()  {
	flag.Parse()

	gb := gbench.New(*url, *concurrent, *requests)
	gb.Run()
}
