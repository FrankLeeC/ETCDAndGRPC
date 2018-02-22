// Author: Frank Lee
// Date: 2017/9/30
// Time: 17:12

package calctest

import (
    "log"
    "net"
    "testing"
    "time"

    "github.com/coreos/etcd/clientv3"
    "github.com/coreos/etcd/clientv3/naming"
    "google.golang.org/grpc"
    gn "google.golang.org/grpc/naming"

    "golang.org/x/net/context"

    //"github.com/heyitsanthony/scraps/calc/calc"
    // GOPATH=~/etcd-vendor/ go test -v
    "etcd/calctest/cacl"
)

var endpoint = "http://192.168.171.136:2379"
var service_name = "the-service"

type calcServer struct{}

func (cs *calcServer) Multiply(ctx context.Context, m *calc.MultiplyRequest) (*calc.MultiplyResponse, error) {
    return &calc.MultiplyResponse{Z: m.X * m.Y}, nil
}

func TestClientTestService(t *testing.T) {
    if err := RegisterService(); err != nil {
        t.Fatal(err)
    }

    gs := grpc.NewServer()
    calc.RegisterCalcServer(gs, &calcServer{})
    ln, lerr := net.Listen("tcp", "127.0.0.1:9999")
    if lerr != nil {
        t.Fatal(lerr)
    }
    go gs.Serve(ln)

    cli, err := clientv3.New(clientv3.Config{
        Endpoints:   []string{endpoint},
        DialTimeout: time.Second * 5,
    })

    if err != nil {
        t.Fatalf("connect etcd (%v)", err)
    }

    defer cli.Close()

    r := &naming.GRPCResolver{Client: cli}
    b := grpc.RoundRobin(r)

    conn, err := grpc.Dial(service_name, grpc.WithBalancer(b), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("dial err (%v)", err)
    }

    defer conn.Close()
    c := calc.NewCalcClient(conn)
    req := calc.MultiplyRequest{1, 2}
    resp, err := c.Multiply(context.Background(), &req)
    if err != nil {
        t.Fatalf("calc err (%v)", err)
    }
    log.Println(resp)
}

func RegisterService() error {
    cli, err := clientv3.New(clientv3.Config{
        Endpoints:   []string{endpoint},
        DialTimeout: time.Second * 5,
    })
    if err != nil {
        return err
    }
    defer cli.Close()

    cli.Delete(cli.Ctx(), service_name)
    r := &naming.GRPCResolver{Client: cli}
    for _, addr := range []string{"127.0.0.1:9999"} {
        service_node := addr
        if err = r.Update(cli.Ctx(), service_name, gn.Update{Op: gn.Add, Addr: service_node}); err != nil {
            return err
        }
        log.Println("register node :", service_name, service_node, err)
    }
    return nil
}
