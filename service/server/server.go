// Author: Frank Lee
// Date: 2017/9/28
// Time: 18:20
package main

import (
    "context"
    "errors"
    "etcd/service/arith"
    "etcd/service/etcd"
    "log"

    "os"
    "os/signal"
    "syscall"

    "net"

    "fmt"

    "encoding/json"

    "github.com/FrankLeeC/Aurora/config"
    "google.golang.org/grpc"
    "google.golang.org/grpc/naming"
    "google.golang.org/grpc/reflection"
)

type Service struct {

}

func (s Service) Add(ctx context.Context, in *arith.Request) (*arith.Response, error) {
    r := arith.Response{Result: in.Dig1 + in.Dig2}
    fmt.Println("current:", in.Count)
    return &r, nil
}

func (s Service) Minus(ctx context.Context, in *arith.Request) (*arith.Response, error) {
    r := arith.Response{Result: in.Dig1 - in.Dig2}
    return &r, nil
}

func (s Service) Prod(ctx context.Context, in *arith.Request) (*arith.Response, error) {
    r := arith.Response{Result: in.Dig1 * in.Dig2}
    return &r, nil
}

func (s Service) Divide(ctx context.Context, in *arith.Request) (*arith.Response, error) {
    if in.Dig2 == 0 {
        return nil, errors.New("divider is zero")
    }
    r := arith.Response{Result: in.Dig1 / in.Dig2}
    return &r, nil
}

var ip = config.GetString("ip")
var port = ":" + config.GetString("port")

func register(service, uri string) {
    clt, _ := etcd.Client()
    lease := etcd.Lease(clt)
    ctx := context.Background()
    rs, err := lease.Grant(ctx, 5)
    if err != nil {
        log.Fatalln("grant err:", err)
        os.Exit(1)
    }
    id := rs.ID
    k := service + "/" + uri
    b, _ := json.Marshal(naming.Update{Op: naming.Add, Addr: uri, Metadata: service + " metadata"})
    clt.Put(ctx, k, string(b))
    lease.KeepAlive(ctx, id)
}

func deRegister(service, uri string) {
    clt, _ := etcd.Client()
    k := service + "/" + uri
    _, err := clt.Delete(context.Background(), k)
    if err != nil {
        log.Fatalln("deRegister err:", err)
    } else {
        log.Println("deRegister success")
    }
}

func main() {
    lis, err := net.Listen("tcp", port)
    if err != nil {
        log.Fatal("create listener err:", err)
    }

    register("add", ip + port)
    register("minus", ip + port)
    register("prod", ip + port)
    register("divide", ip + port)
    fmt.Println("register over")

    ch := make(chan os.Signal, 1)
    signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGQUIT)
    go func() {
        s := <-ch
        log.Printf("receive signal '%v'", s)
        deRegister("add", ip + port)
        deRegister("minus", ip + port)
        deRegister("prod", ip + port)
        deRegister("divide", ip + port)
        os.Exit(1)
    }()

    s := grpc.NewServer()
    arith.RegisterCalculatorServer(s, &Service{})
    reflection.Register(s)
    s.Serve(lis)
}