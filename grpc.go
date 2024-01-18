package groupcache

import (
	"context"
	"net"
	"reflect"
	"sort"
	"sync"

	"github.com/ksong0xd/groupcache/consistenthash"
	"github.com/ksong0xd/groupcache/groupcachepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultReplicas = 50

type GRPCPool struct {
	groupcachepb.UnimplementedGroupCacheServer

	self string

	opts grpcOpt
	s    *grpc.Server

	mu        sync.Mutex
	peersList []string
	peers     *consistenthash.Map
	conns     map[string]*grpcGetter
}

type grpcOpt struct {
	serverOpts []grpc.ServerOption
	listener   net.Listener
	dialOpts   []grpc.DialOption
	replicas   int
	hashFn     consistenthash.Hash
}

type GRPCPoolOption func(opt *grpcOpt)

func WithServerOptions(serverOpts ...grpc.ServerOption) GRPCPoolOption {
	return func(opts *grpcOpt) {
		opts.serverOpts = serverOpts
	}
}

func WithListener(lis net.Listener) GRPCPoolOption {
	return func(opts *grpcOpt) {
		opts.listener = lis
	}
}

func WithDialOptions(dialOpts ...grpc.DialOption) GRPCPoolOption {
	return func(opts *grpcOpt) {
		opts.dialOpts = dialOpts
	}
}

func WithReplicas(replicas int) GRPCPoolOption {
	return func(opts *grpcOpt) {
		opts.replicas = replicas
	}
}

func WithHash(hashFn consistenthash.Hash) GRPCPoolOption {
	return func(opts *grpcOpt) {
		opts.hashFn = hashFn
	}
}

func NewGRPCPool(self string, opts ...GRPCPoolOption) (*GRPCPool, error) {
	pool := &GRPCPool{
		self: self,
		opts: grpcOpt{
			replicas: defaultReplicas,
		},
		conns: map[string]*grpcGetter{},
	}
	for _, opt := range opts {
		opt(&pool.opts)
	}
	pool.peers = consistenthash.New(pool.opts.replicas, pool.opts.hashFn)

	pool.s = grpc.NewServer(pool.opts.serverOpts...)
	pool.s.RegisterService(&groupcachepb.GroupCache_ServiceDesc, pool)

	lis := pool.opts.listener
	if lis == nil {
		// extract the port number.
		_, port, err := net.SplitHostPort(self)
		if err == nil {
			lis, err = net.Listen("tcp", ":"+port)
			if err != nil {
				return nil, err
			}
		} else {
			lis, err = net.Listen("tcp", self)
			if err != nil {
				return nil, err
			}
		}
	}
	go pool.s.Serve(lis)

	RegisterPeerPicker(func() PeerPicker { return pool })
	return pool, nil
}

func (p *GRPCPool) Shutdown() {
	p.s.GracefulStop()
}

func (p *GRPCPool) newListener() (net.Listener, error) {
	if p.opts.listener != nil {
		return p.opts.listener, nil
	}

	_, port, err := net.SplitHostPort(p.self)
	if err != nil {
		return net.Listen("tcp", p.self)
	}
	return net.Listen("tcp", ":"+port)
}

// Set updates the pool's list of peers.
// Each peer value should be an endpoint of gRPC, e.g. "localhost:8080".
func (p *GRPCPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	sort.Strings(peers)
	if reflect.DeepEqual(peers, p.peersList) {
		return
	}
	p.peers = consistenthash.New(p.opts.replicas, p.opts.hashFn)
	p.peers.Add(peers...)
	p.peersList = peers
}

func (p *GRPCPool) PickPeer(key string) (ProtoGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.peers.IsEmpty() {
		return nil, false
	}
	if peer := p.peers.Get(key); peer != p.self {
		c, ok := p.conns[peer]
		if !ok {
			conn, err := grpc.Dial(peer, p.opts.dialOpts...)
			if err != nil {
				// maybe PickPeer should return an error as well.
				// TODO: log the error at least.
				return nil, false
			}
			c = &grpcGetter{groupcachepb.NewGroupCacheClient(conn)}
			p.conns[peer] = c
		}
		return c, true
	}
	return nil, false
}

// Get implements the gRPC method of GroupCacheServer.
func (p *GRPCPool) Get(ctx context.Context, req *groupcachepb.GetRequest) (*groupcachepb.GetResponse, error) {
	group := GetGroup(req.Group)
	if group == nil {
		return nil, status.Errorf(codes.NotFound, "group %s not found", req.Group)
	}

	group.Stats.ServerRequests.Add(1)
	var value []byte
	// Do not just call Get(), as it may forward the request to another server
	// in case that the list of peers has been changed -- it is possible in k8s
	// environment.
	if err := group.get(ctx, req.Key, AllocatingByteSliceSink(&value), false /*usePeers*/); err != nil {
		return nil, err
	}
	return &groupcachepb.GetResponse{
		Value: value,
	}, nil
}

type grpcGetter struct {
	client groupcachepb.GroupCacheClient
}

func (g *grpcGetter) Get(ctx context.Context, in *groupcachepb.GetRequest) (*groupcachepb.GetResponse, error) {
	return g.client.Get(ctx, in)
}
