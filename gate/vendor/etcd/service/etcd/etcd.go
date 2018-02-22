// Author: Frank Lee
// Date: 2017/9/28
// Time: 19:12
package etcd

import (
    "strings"

    "github.com/FrankLeeC/Aurora/config"
    clt "github.com/coreos/etcd/clientv3"
)

var endPoints = config.GetString("etcd")

func Client() (*clt.Client, error) {
    return clt.New(clt.Config{
        Endpoints: strings.Split(endPoints, ","),
    })
}

func Lease(c *clt.Client) clt.Lease {
    return clt.NewLease(c)
}