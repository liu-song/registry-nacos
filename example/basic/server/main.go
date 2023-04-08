// Copyright 2021 CloudWeGo Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/registry-nacos/example/hello/kitex_gen/api"
	"github.com/kitex-contrib/registry-nacos/example/hello/kitex_gen/api/hello"
	"github.com/kitex-contrib/registry-nacos/registry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/shirou/gopsutil/process"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"
)

var (
	Reg = prometheus.NewRegistry()

	PushgatewayUrl = "127.0.0.1:9091"

	CpuMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "Cpu_percent",
		Help: "Cpu ",
	}, []string{"CPU"})

	MemMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "Mem_percent",
		Help: "Mem",
	}, []string{"Mem"})

	GoroutineNUmMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "Goroutine_Num",
		Help: "Goroutine",
	}, []string{"Goroutone_Num"})
)

type HelloImpl struct{}

func (h *HelloImpl) Echo(_ context.Context, req *api.Request) (resp *api.Response, err error) {
	resp = &api.Response{
		Message: req.Message,
	}
	return
}

func main() {

	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)

	}()

	MetricsInit()
	r, err := registry.NewDefaultNacosRegistry()
	if err != nil {
		panic(err)
	}
	svr := hello.NewServer(
		new(HelloImpl),
		server.WithRegistry(r),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "Hello"}),
		server.WithServiceAddr(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}),
	)
	if err := svr.Run(); err != nil {
		log.Println("server stopped with error:", err)
	} else {
		log.Println("server stopped")
	}
}

func MetricsInit() {
	Reg.MustRegister(
		CpuMetric,
		MemMetric,
		GoroutineNUmMetric,
	)
	pusher := push.New(PushgatewayUrl, "nacos").Gatherer(Reg)
	ticker := time.NewTicker(1 * time.Second)

	p, _ := process.NewProcess(int32(os.Getpid()))
	go func() {
		for {
			select {
			case <-ticker.C:
				cpuPercent, err := p.Percent(time.Second)
				if err != nil {
					log.Println(err)
				}
				memPercent, _ := p.MemoryPercent()
				fmt.Printf("CPU usage: %.2f%% ", cpuPercent)
				fmt.Printf("mem usage: %.2f%% ", memPercent)
				fmt.Printf("Number of goroutines: %d\n", runtime.NumGoroutine())
				CpuMetric.With(prometheus.Labels{"CPU": "CPU"}).Set(float64(cpuPercent))
				MemMetric.With(prometheus.Labels{"Mem": "Mem"}).Set(float64(memPercent))
				GoroutineNUmMetric.With(prometheus.Labels{"Goroutone_Num": "Goroutone_Num"}).Set(float64(runtime.NumGoroutine()))
				if err := pusher.Push(); err != nil {
					fmt.Println("Error pushing to Pushgateway:", err)
				}
			}
		}
	}()
}
