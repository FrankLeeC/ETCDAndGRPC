// Author: Frank Lee
// Date: 2017/9/29
// Time: 20:25
package main

import (
    "context"
    "etcd/service/arith"
    "fmt"

    "google.golang.org/grpc"
)


func main() {
    //http.Handle("/add", errorHandler(add))
    //http.Handle("/minus", errorHandler(minus))
    //http.Handle("/prod", errorHandler(prod))
    //http.Handle("/divide", errorHandler(divide))
    //s := &http.Server{
    //    Addr:           ":9090",
    //    Handler:        http.DefaultServeMux,
    //    ReadTimeout:    10 * time.Second,
    //    WriteTimeout:   10 * time.Second,
    //    MaxHeaderBytes: 1 << 20,
    //}
    //go func(){
    //    if err := s.ListenAndServe(); err != nil {
    //        fmt.Println("listen err:", err)
    //    }
    //    log.Println("shutdown")
    //}()
    //c := make(chan os.Signal)
    //signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
    //fmt.Println(<- c)
    //s.Shutdown(context.TODO())
    //time.Sleep(time.Second * 1)

    conn, err := grpc.Dial("192.168.171.136:9090", grpc.WithInsecure())
    if err != nil {
        fmt.Println("dial err:", err)
    }
    r, err := arith.NewCalculatorClient(conn).Add(context.Background(), &arith.Request{Dig1: 10, Dig2: 20, Count: 10})
    if err != nil {
        fmt.Println("err:", err)
    }
    fmt.Println(r.Result)
}
