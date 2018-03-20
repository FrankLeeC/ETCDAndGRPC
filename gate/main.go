// Author: Frank Lee
// Date: 2017/9/29
// Time: 20:25
package main

import (
	"context"
	"encoding/json"
	"etcd/service/arith"
	"etcd/service/etcd"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coreos/etcd/clientv3/naming"
	"google.golang.org/grpc"
)

type errorHandler func(http.ResponseWriter, *http.Request) error

func (h errorHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	err := h(rw, r)
	if err != nil {
		fmt.Fprint(rw, http.StatusInternalServerError, err)
	}
}

var i = 0

var b grpc.Balancer

func init() {
	c, _ := etcd.Client()
	r1 := &naming.GRPCResolver{Client: c}
	b = grpc.RoundRobin(r1)
}

func main() {
	//client.Register()
	http.Handle("/add", errorHandler(add))
	http.Handle("/minus", errorHandler(minus))
	http.Handle("/prod", errorHandler(prod))
	http.Handle("/divide", errorHandler(divide))
	s := &http.Server{
		Addr:           ":9090",
		Handler:        http.DefaultServeMux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		if err := s.ListenAndServe(); err != nil {
			fmt.Println("listen err:", err)
		}
		log.Println("shutdown")
	}()
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println(<-c)
	s.Shutdown(context.TODO())
}

func add(rw http.ResponseWriter, r *http.Request) error {
	fmt.Printf("\n\n\n")
	fmt.Println("---add---")
	var req arith.Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Fprintf(rw, "%d", http.StatusBadRequest)
		return err
	}
	req.Count = int64(i)
	i++
	//c, _ := etcd.Client()
	//r1 := &naming.GRPCResolver{Client: c}
	//b := grpc.RoundRobin(r1)
	con, err := grpc.Dial("add", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithBalancer(b))
	c1 := arith.NewCalculatorClient(con)
	cr, err := c1.Add(context.Background(), &req)
	//c := client.GetClient("add")
	//cr, err := c.Add(context.Background(), &req)
	fmt.Println("---over---")
	if err != nil {
		fmt.Fprint(rw, err)
		return err
	} else {
		fmt.Fprint(rw, cr.Result)
		return nil
	}
}

func minus(rw http.ResponseWriter, r *http.Request) error {
	var req arith.Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Fprintf(rw, "%d", http.StatusBadRequest)
		return err
	}
	req.Count = int64(i)
	i++
	//c, _ := etcd.Client()
	//r1 := &naming.GRPCResolver{Client: c}
	//b := grpc.RoundRobin(r1)
	con, err := grpc.Dial("minus", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithBalancer(b))
	c1 := arith.NewCalculatorClient(con)
	cr, err := c1.Minus(context.Background(), &req)
	if err != nil {
		fmt.Fprint(rw, err)
		return err
	} else {
		fmt.Fprint(rw, cr.Result)
		return nil
	}
}

func prod(rw http.ResponseWriter, r *http.Request) error {
	var req arith.Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Fprintf(rw, "%d", http.StatusBadRequest)
		return err
	}
	req.Count = int64(i)
	i++
	//c, _ := etcd.Client()
	//r1 := &naming.GRPCResolver{Client: c}
	//b := grpc.RoundRobin(r1)
	con, err := grpc.Dial("prod", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithBalancer(b)) // grpc.WithBlock()必须加上，否则找不到地址
	c1 := arith.NewCalculatorClient(con)
	cr, err := c1.Prod(context.Background(), &req)
	if err != nil {
		fmt.Fprint(rw, err)
		return err
	} else {
		fmt.Fprint(rw, cr.Result)
		return nil
	}
}

func divide(rw http.ResponseWriter, r *http.Request) error {
	var req arith.Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Fprintf(rw, "%d", http.StatusBadRequest)
		return err
	}
	req.Count = int64(i)
	i++
	//c, _ := etcd.Client()
	//r1 := &naming.GRPCResolver{Client: c}
	//b := grpc.RoundRobin(r1)
	con, err := grpc.Dial("divide", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithBalancer(b))
	c1 := arith.NewCalculatorClient(con)
	cr, err := c1.Divide(context.Background(), &req)
	if err != nil {
		fmt.Fprint(rw, err)
		return err
	} else {
		fmt.Fprint(rw, cr.Result)
		return nil
	}
}
