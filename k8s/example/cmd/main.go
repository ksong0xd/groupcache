package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jmuk/groupcache"
	"github.com/jmuk/groupcache/k8s"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Assume these are your existing DB handler functions
func getFromDB1(key string) (string, error) {

	//slow down for 1 seconds
	time.Sleep(1 * time.Second)

	klog.Info("getFromDB1 called!!!")
	return "db result 1", nil
}

func getFromDB2(key string) (string, error) {

	//slow down for 1 seconds
	time.Sleep(1 * time.Second)

	// Your DB access logic here
	klog.Info("getFromDB2 called!!!")

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

	var serviceName string
	var namespace string
	var self string
	var gcPortStr string
	var portStr string

	//if dev is false then use k8 config
	if os.Getenv("dev") != "true" {
		serviceName = os.Getenv("SERVICE_NAME")
		if serviceName == "" {
			panic("SERVICE_NAME is not set")
		}
		namespace = os.Getenv("NAMESPACE")
		if namespace == "" {
			panic("NAMESPACE is not set")
		}
		self = os.Getenv("SELF")
		if self == "" {
			panic("SELF is not set")
		}
		gcPortStr = os.Getenv("GROUPCACHE_PORT")
		if gcPortStr == "" {
			panic("GROUPCACHE_PORT is not set")
		}
		gcPort, err := strconv.ParseInt(gcPortStr, 10, 32)
		if err != nil {
			panic(err)
		}
		portStr = os.Getenv("HTTP_PORT")
		if portStr == "" {
			panic("HTTP_PORT is not set")
		}

		restClient, err := rest.InClusterConfig()
		if err != nil {
			panic(err)
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
	} else {

		//may need to run kubernetes proxy
		os.Setenv("KUBERNETES_SERVICE_HOST", "127.0.0.1")
		os.Setenv("KUBERNETES_SERVICE_PORT", "8001")

		//if dev is true then use local config
		serviceName = "fib"
		namespace = "default"
		self = "127.0.0.1"
		gcPortStr = "8081"
		portStr = "8080"
	}

	var g *groupcache.Group
	getter := groupcache.GetterFunc(func(ctx context.Context, key string, sink groupcache.Sink) error {
		klog.Info("getter called")
		klog.Infof("new data saved to: %s(self), key: %s", self, key)

		klog.Info("get context")
		var result string

		eg, ctx := errgroup.WithContext(ctx)

		eg.Go(func() error {

			var tmpResult string

			//get http request from context ctx
			klog.Info("get http request context")

			//list up all ctx values
			//httpCtx := ctx.Value("http.request")
			//httpCtx := ctx.Value("api.url")

			resultJson := make(map[string]interface{})
			json.Unmarshal([]byte(key), &resultJson)

			if resultJson != nil {
				//r := httpCtx.(*http.Request)
				//if r == nil {
				//	klog.Info("http request is nil")
				//	return nil
				//}

				//get db handler from map
				//klog.Info("get db handler with " + r.URL.Path)
				//dbHandler := dbHandlers[r.URL.Path]

				dbHandler := dbHandlers[resultJson["url"].(string)]
				value, err := dbHandler(key)
				if err != nil {
					return err
				}

				klog.Infof("key: %s, value: %d", key, value)
				tmpResult = "key-value-for-" + value
			} else {
				klog.Info("http request context is nil for the case of peer call to sharding a data")
				//sharding feature is that the data is sharded by key and actual process is distributed
				//so there is case that getter handler is called by peer and the case it doesn't have http request context
				//the sharding data is not necessary for the case of API + DB case
				//because the data is already sharded by API which is being called by router (ingress)
				//so let's turn off sharding feature if it's possible.

				tmpResult = "N/A"
			}

			//put tmpResult into %result in atomic way
			result = tmpResult

			return nil
		})
		if err := eg.Wait(); err != nil {
			return err
		}

		return sink.SetString(result)
	})
	g = groupcache.NewGroup("fib", 1024*1024, getter)

	http.Handle("/handler1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "key is not specified", http.StatusInternalServerError)
			return
		}

		//create a map for json
		datas := map[string]string{
			"key": key,
			"url": r.URL.Path,
		}

		jsonString, err := json.Marshal(datas)

		// get context from http.request
		//httpContext := context.WithValue(r.Context(), "http.request", r)
		//httpContext := context.Background()
		//httpContext := context.WithValue(ctx, "api.url", "/handler1")
		c := context.WithValue(context.Background(), "key1", "value1")
		c = context.WithValue(c, "key2", "value2")

		var v string
		err = g.Get(r.Context(), string(jsonString), groupcache.StringSink(&v))
		if err != nil {
			http.Error(w, "failed to obtain the result", http.StatusInternalServerError)
			return
		}

		w.Header().Add("content-type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(v))
	}))
	err := http.ListenAndServe(":"+portStr, nil)
	panic(err)
}
