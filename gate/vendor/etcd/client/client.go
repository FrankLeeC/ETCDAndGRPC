// Author: Frank Lee
// Date: 2017/9/28
// Time: 19:31
package client

import (
    "etcd/service/arith"

    "etcd/service/etcd"

    "fmt"

    "github.com/FrankLeeC/Aurora/log"
    "github.com/coreos/etcd/clientv3/naming"
    "google.golang.org/grpc"
)

var (
    b grpc.Balancer
    m = make(map[string]arith.CalculatorClient)
)

func Register() {
    if t, err := createBalance(); err == nil {
        b = t
        createCalculatorClient("add")
        createCalculatorClient("minus")
        createCalculatorClient("prod")
        createCalculatorClient("divide")
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

func createCalculatorClient(service string) {
    con, err := grpc.Dial(service, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithBalancer(b))
    if err != nil {
        log.Fatalln("dial err: %s", err.Error())
    }
    m[service] = arith.NewCalculatorClient(con)
}

func GetClient(service string) arith.CalculatorClient {
    if v, c := m[service]; c {
        return v
    }
    return nil
}
