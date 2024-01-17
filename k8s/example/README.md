# k8 basic

### entire context list

kubectl config get-contexts

### set default context and name space

kubectl config use-context docker-desktop

### create a namespace and set it as default

kubectl create namespace ksong
kubectl config set-context --current --namespace=ksong
kubectl config set-context --current --namespace=default

### get k8 capacity

kubectl top nodes
kubectl describe nodes

### entire node list

kubectl get nodes

### deploy app

kubectl create deployment kubernetes-bootcamp --image=gcr.io/google-samples/kubernetes-bootcamp:v1
kubectl create deployment cache-client --image=localhost:5001/cache-client

### list deployment

kubectl get deployments

### check deployment status

kubectl rollout status deployment/kubernetes-bootcamp
kubectl rollout status deployments

### describe deployment

kubectl describe deployment
kubectl describe deployments/kubernetes-bootcamp
kubectl describe deployments/fib

### remove deployment

kubectl delete deployment kubernetes-bootcamp
kubectl delete deployments/fib

### list pods

kubectl get pods

### read/set name of pod

export POD_NAME="$(kubectl get pods -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')";echo
Name of the Pod: $POD_NAME

### log pod

kubectl logs $POD_NAME

### get env of pod

kubectl exec "$POD_NAME" -- env

### ssh to pod

kubectl exec -ti $POD_NAME -- bash

### run proxy

kubectl proxy

### get services

kubectl get services

### service details

kubectl describe services/kubernetes-bootcamp
kubectl describe services/fib

### create a service

kubectl expose deployment/kubernetes-bootcamp --type="NodePort" --port 8080
kubectl expose deployment/fib --type="NodePort" --port 8080

### delete a service

kubectl delete service -l app=kubernetes-bootcamp
kubectl delete service -l app=peer-aware-groupcache

### export port

export NODE_PORT="$(kubectl get services/kubernetes-bootcamp -o go-template='{{(index .spec.ports 0).nodePort}}')"
;echo "NODE_PORT=$NODE_PORT"
export NODE_PORT="$(kubectl get services/helm-chart-1704764238-peer-aware-groupcache -o go-template='{{(index
.spec.ports 0).nodePort}}')";echo "NODE_PORT=$NODE_PORT"

### test service without proxy

curl http://localhost:$NODE_PORT

### use a label

kubectl get pods -l app=kubernetes-bootcamp
kubectl get services -l app=kubernetes-bootcamp
kubectl label pods "$POD_NAME" version=v1
kubectl describe pods "$POD_NAME"
kubectl get pods -l version=v1

### debug

kubectl describe pods fib-6ff8bffbcc-dcjzg
kubectl logs "$POD_NAME"  --all-containers
kubectl get events
kubectl get events --field-selector involvedObject.name=fib-6ff8bffbcc-b9nmz
kubectl describe deployment fib

### see replica set

kubectl get rs

### scale up

kubectl scale deployments/kubernetes-bootcamp --replicas=4
kubectl scale deployments/fib --replicas=1

### get pods with more info

kubectl get pods -o wide

### upgrade image (rolling update including kill old and start new)

kubectl set image deployments/kubernetes-bootcamp kubernetes-bootcamp=jocatalin/kubernetes-bootcamp:v2
kubectl set image deployments/fib fib=localhost:5001/cache-client:latest

### rollout restart deployment

kubectl rollout restart deployment/fib

### rollout undo

kubectl rollout undo deployments/kubernetes-bootcamp

### create pods using yaml file

kubectl create -f fib.yaml

### how to run open local registry

docker run -d -p 5001:5000 --name registry registry:2.7
docker tag cache-client:latest localhost:5001/cache-client

### build and redeploy with using registry
docker build --tag localhost:5001/cache-client:v24 .
docker push localhost:5001/cache-client:v24
kubectl set image deployments/fib fib=localhost:5001/cache-client:v24

### build and redeploy without using registry (fastest)
docker build --tag cache-client:v40 .
kubectl set image deployments/fib fib=cache-client:v40



### delete/recreate deployment from yaml file to apply latest
kubectl delete -f k8s/example/fib.yaml
kubectl create -f k8s/example/fib.yaml


### create busybox
kubectl run curl-ksong --image=radial/busyboxplus:curl -i --tty --rm

### curl per pods

curl http://localhost:8001/api/v1/namespaces/default/pods/fib-6db7f6c4b8-4ntxk:8080/proxy/

# distributed cache

### modified version of groupCache

cache data stored by using shard (data will be spread)

### git

main: https://github.com/golang/groupcache
others:
https://github.com/mailgun/groupcache
https://github.com/vimeo/galaxycache
https://github.com/orijtech/groupcache/tree/instrument-with-opencensus

### alternative

https://github.com/buraksezer/olric

### sql level cache with reddis

https://github.com/zeromicro/go-zero

### peer-aware-groupcache

https://github.com/robwil/peer-aware-groupcache

### with k8 (*)

https://github.com/jmuk/groupcache
https://github.com/udhos/kubegroup

https://github.com/jmuk/groupcache
https://github.com/danielvegamyhre/minicache
https://github.com/allegro/bigcache
https://github.com/orijtech/groupcache

### posts (why)

https://giedrius.blog/2022/03/04/distributed-systems-magic-groupcache-in-thanos/
https://www.mailgun.com/blog/it-and-engineering/golangs-superior-cache-solution-memcached-redis/?utm_source=pocket_reader
https://medium.com/codex/our-go-cache-library-choices-406f2662d6b
https://joshua.themarshians.com/post/doozer-groupcache/?utm_source=pocket_reader

### posts (tech)

https://medium.com/orijtech-developers/groupcache-instrumented-by-opencensus-6a625c3724c

### peer watch

$ helm install -n peer-aware-groupcache helm-chart/
$ kubectl scale --replicas=3 deployment/peer-aware-groupcache
$ export NODE_PORT=$(kubectl get --namespace default -o jsonpath="{.spec.ports[0].nodePort}" services
peer-aware-groupcache)
$ export NODE_IP=$(kubectl get nodes --namespace default -o jsonpath="{.items[0].status.addresses[0].address}")
$ for i in `seq 0 9`; do echo 1234$i; curl http://$NODE_IP:$NODE_PORT/factors?n=1234$i; done



