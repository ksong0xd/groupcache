//package main
//
//import (
//	"context"
//	"fmt"
//	"github.com/jmuk/groupcache"
//	"github.com/jmuk/groupcache/k8s"
//	"k8s.io/client-go/kubernetes"
//	"k8s.io/client-go/rest"
//	"k8s.io/klog/v2"
//	"net/http"
//	"os"
//	"strconv"
//)
//
//func main() {
//	klog.Infof("starting")
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	restClient, err := rest.InClusterConfig()
//	if err != nil {
//		panic(err)
//	}
//
//	serviceName := os.Getenv("SERVICE_NAME")
//	if serviceName == "" {
//		panic("SERVICE_NAME is not set")
//	}
//	namespace := os.Getenv("NAMESPACE")
//	if namespace == "" {
//		panic("NAMESPACE is not set")
//	}
//	self := os.Getenv("SELF")
//	if self == "" {
//		panic("SELF is not set")
//	}
//	gcPortStr := os.Getenv("GROUPCACHE_PORT")
//	if gcPortStr == "" {
//		panic("GROUPCACHE_PORT is not set")
//	}
//	gcPort, err := strconv.ParseInt(gcPortStr, 10, 32)
//	if err != nil {
//		panic(err)
//	}
//	portStr := os.Getenv("HTTP_PORT")
//	if portStr == "" {
//		panic("HTTP_PORT is not set")
//	}
//
//	m, err := k8s.NewPeersManager(
//		ctx,
//		kubernetes.NewForConfigOrDie(restClient),
//		serviceName,
//		namespace,
//		int(gcPort),
//		fmt.Sprintf("%s:%d", self, gcPort),
//	)
//	if err != nil {
//		panic(err)
//	}
//	defer m.Stop()
//
//	var g *groupcache.Group
//	getter := groupcache.GetterFunc(
//		func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
//			// Retrieve the function handler from the context and execute it.
//			if handler, ok := ctx.Value("handler").(func()); ok {
//				handler()
//			}
//
//			// ... (rest of your getter function code)
//
//			//set string hello to dest
//			return dest.SetString("echo :" + key)
//		})
//
//	g = groupcache.NewGroup("fib", 1024*1024, getter)
//
//	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//
//		key := r.URL.Query().Get("q")
//		if key == "" {
//			http.Error(w, "q is not specified", http.StatusInternalServerError)
//			return
//		}
//
//		var v string
//
//		// Create a function handler.
//		handler := func() {
//			fmt.Println("handler called " + key)
//		}
//
//		requestContext := context.WithValue(r.Context(), "handler", handler)
//
//		err := g.Get(requestContext, key, groupcache.StringSink(&v))
//		if err != nil {
//			http.Error(w, "failed to obtain the result", http.StatusInternalServerError)
//			return
//		}
//
//		w.Header().Add("content-type", "text/plain")
//		w.WriteHeader(http.StatusOK)
//		w.Write([]byte(v))
//	}))
//	err = http.ListenAndServe(":"+portStr, nil)
//	panic(err)
//}
