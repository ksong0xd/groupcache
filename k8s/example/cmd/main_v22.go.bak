package main

import (
	"context"
	"fmt"
	"github.com/jmuk/groupcache"
	"github.com/jmuk/groupcache/k8s"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"net/http"
	"os"
	"strconv"
)

// Assume these are your existing DB handler functions
func getFromDB1(key string) (string, error) {
	// Your DB access logic here
	klog.Info("getFromDB1 called")
	return "db result 1", nil
}

func getFromDB2(key string) (string, error) {
	// Your DB access logic here
	klog.Info("getFromDB2 called")
	return "db result 2", nil
}

// Map of DB handlers
var dbHandlers = map[string]func(string) (string, error){
	"/handler1": getFromDB1,
	"/handler2": getFromDB2,
}

func main() {
	klog.Infof("starting")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	restClient, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}

	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		panic("SERVICE_NAME is not set")
	}
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		panic("NAMESPACE is not set")
	}
	self := os.Getenv("SELF")
	if self == "" {
		panic("SELF is not set")
	}
	gcPortStr := os.Getenv("GROUPCACHE_PORT")
	if gcPortStr == "" {
		panic("GROUPCACHE_PORT is not set")
	}
	gcPort, err := strconv.ParseInt(gcPortStr, 10, 32)
	if err != nil {
		panic(err)
	}
	portStr := os.Getenv("HTTP_PORT")
	if portStr == "" {
		panic("HTTP_PORT is not set")
	}

	m, err := k8s.NewPeersManager(
		ctx,
		kubernetes.NewForConfigOrDie(restClient),
		serviceName,
		namespace,
		int(gcPort),
		fmt.Sprintf("%s:%d", self, gcPort),
	)
	if err != nil {
		panic(err)
	}
	defer m.Stop()

	var g *groupcache.Group
	getter := groupcache.GetterFunc(func(ctx context.Context, key string, sink groupcache.Sink) error {
		klog.Info("getter called")
		klog.Infof("new data saved to: %s(self), key: %s", self, key)

		klog.Info("get context")
		var result string
		//_, ctx = errgroup.WithContext(ctx)

		//klog.Info("read data from cache")
		//var s string
		//if err := g.Get(ctx, key, groupcache.StringSink(&s)); err != nil {
		//	klog.Info(err)
		//	return err
		//}

		//get http request from context ctx
		klog.Info("get http request context")

		//list up all ctx values
		r := ctx.Value("http.request").(*http.Request)

		if r == nil {
			klog.Info("http request is nil")
			return nil
		}

		klog.Info("get db handler with " + r.URL.Path)
		dbHandler := dbHandlers[r.URL.Path]
		value, err := dbHandler(key)
		if err != nil {
			return err
		}

		klog.Infof("key: %s, value: %d", key, value)

		result = "key-value-for-" + value
		return sink.SetString(result)
	})
	g = groupcache.NewGroup("fib", 1024*1024, getter)

	http.Handle("/handler1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "key is not specified", http.StatusInternalServerError)
			return
		}
		var v string
		err := g.Get(r.Context(), key, groupcache.StringSink(&v))
		if err != nil {
			http.Error(w, "failed to obtain the result", http.StatusInternalServerError)
			return
		}
		w.Header().Add("content-type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(v))
	}))
	err = http.ListenAndServe(":"+portStr, nil)
	panic(err)
}
