export TOKEN=token-01
export CLUSTER_STATE=new
export NAME_1=machine-1
export NAME_2=machine-2
export NAME_3=machine-3
export HOST_1=192.168.171.136
export HOST_2=192.168.171.137
export HOST_3=192.168.171.138
export CLUSTER=${NAME_1}=http://${HOST_1}:2380,${NAME_2}=http://${HOST_2}:2380,${NAME_3}=http://${HOST_3}:2380

THIS_NAME=${NAME_1}
THIS_IP=${HOST_1}
nohup $ETCD/etcd --data-dir=data.etcd --name ${THIS_NAME} \
        --initial-advertise-peer-urls http://${THIS_IP}:2380 --listen-peer-urls http://${THIS_IP}:2380 \
        --advertise-client-urls http://${THIS_IP}:2379 --listen-client-urls http://${THIS_IP}:2379 \
        --initial-cluster ${CLUSTER} \
        --initial-cluster-state ${CLUSTER_STATE} --initial-cluster-token ${TOKEN} >> $ETCD/etcd.log &
