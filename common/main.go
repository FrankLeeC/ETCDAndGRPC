// Author: Frank Lee
// Date: 2017/9/22
// Time: 10:58
package main

import (
    "context"
    "fmt"
    "strconv"
    "time"

    "github.com/coreos/etcd/clientv3"
    "github.com/coreos/etcd/clientv3/concurrency"
)

func main() {
	//testTxn()
	//testWatch()
    //testTTL()
    //testLeaseTimeout()
    //testLeaseTTL()
    //testLock()
    //testRevision()
    //testMulti()
    ranges()
    //get("add/")
    getWithPrefix("add/")
    //deleteRange()
}

func testTxn() {
	//txn(true)
	//txn(false)
}

func txn(success bool) {
	if success {
		put("key111", strconv.FormatBool(success))
	} else {
		del("key111")
	}
	cli := getClient()
	if cli != nil {
		defer cli.Close()
		txn := cli.Txn(context.TODO())
		resp, err := txn.If(
			clientv3.Compare(clientv3.Value("key111"), "=", "true")).Then(
			clientv3.OpPut("key111", "yyy"), clientv3.OpPut("key222", "nnn")).Else(
			clientv3.OpPut("key333", "yuy"), clientv3.OpPut("key444", "poi")).Commit()
		if err != nil {
			fmt.Println("txn err:", err)
		} else {
			fmt.Printf("header : %s, response : %s, success : %s\n", resp.Header, resp.Responses, resp.Succeeded)
		}
		get("key111")
		get("key222")
		fmt.Println("-----------")
		get("key333")
		get("key444")
	}
}

func testDelete() {
	put("key109", "val109")
	fmt.Println("---put over---")
	get("key109")
	fmt.Println("---get over---")
	del("key109")
	fmt.Println("---delete over---")
	get("key109")
}

func del(key string) {
	cli := getClient()
	if cli != nil {
		defer cli.Close()
		resp, err := cli.Delete(context.Background(), key)
		if err != nil {
			fmt.Println("delete err:", err)
		}
		fmt.Println(resp.Header)
		fmt.Println(resp.PrevKvs)
	}
}

func testWatch() {
	go func() {
		time.Sleep(time.Second * 3)
		fmt.Println("-------put-------")
		for i := 0; i < 10; i++ {
			put("key101", "val"+strconv.Itoa(101+i))
		}
	}()
	watcher("key101")
}

func watcher(key string) {
	cli := getClient()
	if cli != nil {
		defer cli.Close()
		resp := cli.Watch(context.Background(), key)
		for r := range resp {
			for _, ev := range r.Events {
				fmt.Printf("in watcher: %s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
			}
		}
	}
}

func testMulti() {
    put("service/1.2.3.4", "123")
    put("service/11.22.33.44", "890")
    put("service/111.222.11.11", "6678")
    getWithPrefix("service")
}

func put(key, value string) {
	cli := getClient()
	if cli != nil {
		defer cli.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		resp, err := cli.Put(ctx, key, value)
		cancel()
		if err != nil {
			fmt.Println("put err:", err)
		} else {
			fmt.Println(resp.Header)
		}
	}
}

func get(key string) {
	cli := getClient()
	if cli != nil {
		defer cli.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		resp, err := cli.Get(ctx, key)
		cancel()
		if err != nil {
			fmt.Println("get err:", err)
		} else {
			for _, ev := range resp.Kvs {
				fmt.Printf("%s : %s\n", ev.Key, ev.Value)
			}
		}
	}
}

func getWithPrefix(key string) {
    cli := getClient()
    if cli != nil {
        defer cli.Close()
        ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
        resp, err := cli.Get(ctx, key, clientv3.WithPrefix())
        cancel()
        if err != nil {
            fmt.Println("get err:", err)
        } else {
            for _, ev := range resp.Kvs {
                fmt.Printf("%s : %s\n", ev.Key, ev.Value)
            }
        }
    }
}

func ranges() {
    cli := getClient()
    if cli != nil {
        defer cli.Close()
        resp, _ := cli.Get(context.Background(), "0", clientv3.WithRange("z"))
        for _, kv := range resp.Kvs {
            fmt.Println(string(kv.Key), string(kv.Value))
        }
    }
}

func getClient() *clientv3.Client {
	cli, err := clientv3.New(clientv3.Config{
		//Endpoints: []string{"docker1:2379"},
		//Endpoints: []string{"docker2:2379"},
		//Endpoints: []string{"docker3:2379"},
		Endpoints: []string{"docker1:2379", "docker2:2379", "docker3:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		fmt.Println("dial err:", err)
		return nil
	}
	return cli
}

//func testTTL() {
//    cli1 := getClient()
//    cli2 := getClient()
//    if cli1 != nil && cli2 != nil {
//        defer cli1.Close()
//        defer cli2.Close()
//        cli1.Grant()
//    }
//}

func testTTL() {
    cli := getClient()
    //go func(){
    //    for {
    //        watcher("123")
    //    }
    //}()
    if cli != nil {
        get("plm")
        fmt.Println("--first get---")
        defer cli.Close()
        ctx := context.Background()
        lease := clientv3.NewLease(cli)
        r, _ := lease.Grant(ctx, 1)
        lid := r.ID
        cli.Put(ctx, "plm", "qqq", clientv3.WithLease(lid))
        get("plm")
        fmt.Println("--second get---")
        lease.KeepAliveOnce(ctx, lid)
        lease.Revoke(ctx, lid)
        lease.Close()
        time.Sleep(time.Second * 3)
        get("plm")
        fmt.Println("--third get---")
    }
}

func testLeaseTimeout() {
    del("lease")
    fmt.Println("delete over")
    cli := getClient()
    go func(){
        for {
            watcher("lease")
        }
    }()
    if cli != nil {
        defer cli.Close()
        ctx := context.Background()
        lease := clientv3.NewLease(cli)
        r, _ := lease.Grant(ctx, 1)
        lid := r.ID
        cli.Put(ctx, "lease", "qqq", clientv3.WithLease(lid))
        get("lease")
        fmt.Println("---get---")
        lease.KeepAliveOnce(ctx, lid)
        time.Sleep(time.Second * 5)
        fmt.Println("sleep over")
        lease.Revoke(ctx, lid)
    }
    select {

    }
}

func testLeaseTTL() {
    del("revoke")
    fmt.Println("delete over")
    cli := getClient()
    go func(){
        i := 0
        for {
            //watcher("revoke")
            get("revoke")
            fmt.Println("-----get over----", i)
            i++
            time.Sleep(time.Second * 2)
        }
    }()
    if cli != nil {
        defer cli.Close()
        ctx := context.Background()
        lease := clientv3.NewLease(cli)
        r, _ := lease.Grant(ctx, 4)
        lid := r.ID
        cli.Put(ctx, "revoke", "qqq", clientv3.WithLease(lid))
        //for i := 0; i < 10; i++{
        //    lease.KeepAliveOnce(ctx, lid)
        //    fmt.Println("keep alive once")
        //    time.Sleep(time.Second * 3)
        //}
        lease.KeepAlive(ctx, lid)
        fmt.Println("---over---")
        //get("revoke")
        //fmt.Println("---get---")
        //time.Sleep(time.Second * 2)
        //fmt.Println("sleep over")
        //lease.Revoke(ctx, lid)
    }
    select {

    }
}

// output:
// m1 prepare to acquire lock
// m2 prepare to acquire lock
// m2 acquire lock
// m2 release lock
// m1 acquire lock
// m1 release lock
func testLock() {
    go func() {
        cli := getClient()
        session, _ := concurrency.NewSession(cli)
        m := concurrency.NewMutex(session, "lock")
        fmt.Println("m1 prepare to acquire lock")
        m.Lock(context.TODO())
        fmt.Println("m1 acquire lock")
        time.Sleep(time.Second * 3)
        m.Unlock(context.TODO())
        fmt.Println("m1 release lock")
    }()

    go func() {
        cli := getClient()
        session, _ := concurrency.NewSession(cli)
        m := concurrency.NewMutex(session, "lock")
        fmt.Println("m2 prepare to acquire lock")
        m.Lock(context.TODO())
        fmt.Println("m2 acquire lock")
        time.Sleep(time.Second * 3)
        m.Unlock(context.TODO())
        fmt.Println("m2 release lock")
    }()

    select {

    }
}

func testRevision() {
    //put("a", "g")
    cli := getClient()
    rsp, _ := cli.Get(context.Background(), "a", /*clientv3.WithPrefix(), *//*clientv3.WithLastRev()...*/ clientv3.WithMinModRev(1))
    fmt.Printf("%v\n", rsp.Kvs)
}

func deleteRange() {
    cli := getClient()
    if cli != nil {
        defer cli.Close()
        cli.Delete(context.Background(), "0", clientv3.WithRange("z"))
    }
    ranges()
}