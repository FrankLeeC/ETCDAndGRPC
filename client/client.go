// Author: Frank Lee
// Date: 2017/9/28
// Time: 19:31
package client

import (
    "etcd/service/arith"

    "etcd/service/etcd"

    "sync"

    "fmt"

    "github.com/FrankLeeC/Aurora/log"
    "github.com/coreos/etcd/clientv3/naming"
    "google.golang.org/grpc"
)

var (
    b grpc.Balancer
    m = make(map[string]arith.CalculatorClient)
)

func init() {
    if t, err := createBalance(); err == nil {
        b = t
    } else {
        fmt.Println("create balance err:", err)
    }
}

func createBalance() (grpc.Balancer, error) {
    cli, err := etcd.Client()
    if err != nil {
        log.Fatalln("get etcd gate err: %s", err.Error())
        return nil, err
    }
    r := &naming.GRPCResolver{Client: cli}
    return grpc.RoundRobin(r), nil
}

func getCalculatorClient(service string) (arith.CalculatorClient, error) {
    con, err := grpc.Dial(service, grpc.WithInsecure(), grpc.WithBalancer(b))
    if err != nil {
        log.Fatalln("dial err: %s", err.Error())
        return nil, err
    }
    return arith.NewCalculatorClient(con), nil
}

func GetClient(service string) arith.CalculatorClient {
    new(sync.Once).Do(func(){
        if c, err := getCalculatorClient(service); err == nil {
            m[service] = c
        }
    })
    return m[service]
}
