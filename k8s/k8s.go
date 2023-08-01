// package k8s provides the utility to use groupcache
// under kubernetes. See README.md for the details.
package k8s

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/jmuk/groupcache"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

var addressTypes []discoveryv1.AddressType = []discoveryv1.AddressType{
	discoveryv1.AddressTypeIPv4,
	discoveryv1.AddressTypeIPv6,
	discoveryv1.AddressTypeFQDN,
}

type PeersManager struct {
	client      kubernetes.Interface
	serviceName string
	port        int
	pool        *groupcache.GRPCPool
	cancel      context.CancelFunc

	mu    sync.Mutex
	peers map[discoveryv1.AddressType]map[string][]string
}

func NewPeersManager(
	ctx context.Context,
	client kubernetes.Interface,
	serviceName string,
	namespace string,
	port int,
	self string,
	opts ...groupcache.GRPCPoolOption,
) (*PeersManager, error) {
	client.DiscoveryV1().EndpointSlices(namespace).Watch(ctx, metav1.ListOptions{})
	if len(opts) == 0 {
		opts = []groupcache.GRPCPoolOption{
			groupcache.WithServerOptions(grpc.Creds(insecure.NewCredentials())),
			groupcache.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
		}
	}
	pool, err := groupcache.NewGRPCPool(self, opts...)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)

	pm := &PeersManager{
		pool:        pool,
		port:        port,
		client:      client,
		serviceName: serviceName,
		cancel:      cancel,
		peers:       map[discoveryv1.AddressType]map[string][]string{},
	}

	pm.watch(ctx, namespace)
	return pm, nil
}

func (pm *PeersManager) Stop() {
	pm.cancel()
	pm.pool.Shutdown()
}

func (pm *PeersManager) watch(ctx context.Context, namespace string) {
	i := informers.NewSharedInformerFactoryWithOptions(
		pm.client,
		time.Minute,
		informers.WithNamespace(namespace),
	)
	i.Discovery().V1().EndpointSlices().Informer().AddEventHandler(pm)
	i.Start(ctx.Done())
}

func (pm *PeersManager) updatePeers() {
	var peers []string
	for _, at := range addressTypes {
		m, ok := pm.peers[at]
		if !ok {
			continue
		}
		for _, ps := range m {
			for _, p := range ps {
				peers = append(peers, fmt.Sprintf("%s:%d", p, pm.port))
			}
		}
	}
	sort.Strings(peers)
	pm.pool.Set(peers...)
}

func (pm *PeersManager) isRelatedEndpointSlice(es *discoveryv1.EndpointSlice) bool {
	serviceName, ok := es.Labels[discoveryv1.LabelServiceName]
	if !ok {
		// service name is not known; not our target.
		return false
	}
	return serviceName == pm.serviceName
}

func (pm *PeersManager) handleEndpointSlice(es *discoveryv1.EndpointSlice) {
	if !pm.isRelatedEndpointSlice(es) {
		return
	}
	pm.mu.Lock()
	defer pm.mu.Unlock()

	peers := make([]string, 0, len(es.Endpoints))
	for _, e := range es.Endpoints {
		// Ignore terminating endpoints.
		if e.Conditions.Terminating != nil && *e.Conditions.Terminating {
			continue
		}
		// Ignore ones not ready.
		if e.Conditions.Ready == nil || !(*e.Conditions.Ready) {
			continue
		}
		if len(e.Addresses) == 0 {
			continue
		}
		peers = append(peers, e.Addresses[0])
	}
	ess, ok := pm.peers[es.AddressType]
	if !ok {
		ess = map[string][]string{}
		pm.peers[es.AddressType] = ess
	}
	ess[es.Name] = peers
	pm.updatePeers()
}

func (pm *PeersManager) OnAdd(obj any, isInInitialList bool) {
	es, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		return
	}
	pm.handleEndpointSlice(es)
}

func (pm *PeersManager) OnUpdate(oldObj, newObj any) {
	es, ok := newObj.(*discoveryv1.EndpointSlice)
	if !ok {
		// Shouldn't happen.
		return
	}
	pm.handleEndpointSlice(es)
}

func (pm *PeersManager) OnDelete(obj any) {
	es, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		return
	}
	if !pm.isRelatedEndpointSlice(es) {
		return
	}
	pm.mu.Lock()
	defer pm.mu.Unlock()

	m, ok := pm.peers[es.AddressType]
	if !ok {
		return
	}
	delete(m, es.Name)
	pm.updatePeers()
}
