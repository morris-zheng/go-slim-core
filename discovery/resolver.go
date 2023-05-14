package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/morris-zheng/go-slim-core/logger"

	clientV3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
)

func Scheme(service string) string {
	return fmt.Sprintf("%s:///%s", schemeName, service)
}

type Resolver struct {
	endpoints  []string
	prefix     string
	client     *clientV3.Client
	logger     logger.Logger
	address    []resolver.Address
	clientConn resolver.ClientConn
	watchChan  clientV3.WatchChan
}

func NewResolver(opt Option, l *logger.Logger) *Resolver {
	schemeName = opt.Prefix
	return &Resolver{
		endpoints: opt.Endpoints,
		logger:    *l,
	}
}

func (r *Resolver) Scheme() string {
	return schemeName
}

func (r *Resolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r.clientConn = cc
	r.prefix = fmt.Sprintf("/%s%s", schemeName, target.URL.Path)

	var err error
	r.client, err = clientV3.New(clientV3.Config{
		Endpoints:   r.endpoints,
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		return nil, err
	}

	if err = r.sync(); err != nil {
		return nil, err
	}

	go r.watch()

	return r, nil
}

func (r *Resolver) ResolveNow(o resolver.ResolveNowOptions) {}

func (r *Resolver) Close() {}

func (r *Resolver) watch() {
	ticker := time.NewTicker(time.Minute)
	r.watchChan = r.client.Watch(context.Background(), r.prefix, clientV3.WithPrefix())

	for {
		select {
		case res, ok := <-r.watchChan:
			if ok {
				r.update(res.Events)
			}
		case <-ticker.C:
			if err := r.sync(); err != nil {
				r.logger.Error(context.Background(), "sync failed", err)
			}
		}
	}
}

func (r *Resolver) update(events []*clientV3.Event) {
	var node Node
	for _, ev := range events {
		switch ev.Type {
		case clientV3.EventTypePut:
			err := node.Decode(ev.Kv.Value)
			if err != nil {
				continue
			}

			addr := resolver.Address{Addr: fmt.Sprintf("%s:%d", node.Host, node.Port), ServerName: node.Name}
			if !Exist(r.address, addr) {
				r.address = append(r.address, addr)
				r.clientConn.UpdateState(resolver.State{Addresses: r.address})
			}
		case clientV3.EventTypeDelete:
			err := node.Decode(ev.Kv.Value)
			if err != nil {
				continue
			}

			addr := resolver.Address{Addr: fmt.Sprintf("%s:%d", node.Host, node.Port)}
			if s, ok := Remove(r.address, addr); ok {
				r.address = s
				r.clientConn.UpdateState(resolver.State{Addresses: r.address})
			}
		}
	}
}

func (r *Resolver) sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := r.client.Get(ctx, r.prefix, clientV3.WithPrefix())
	if err != nil {
		return err
	}

	r.address = []resolver.Address{}
	var node Node

	for _, v := range res.Kvs {
		err := node.Decode(v.Value)
		if err != nil {
			continue
		}

		addr := resolver.Address{Addr: fmt.Sprintf("%s:%d", node.Host, node.Port)}
		r.address = append(r.address, addr)
	}

	r.clientConn.UpdateState(resolver.State{Addresses: r.address})
	return nil
}

func Exist(l []resolver.Address, addr resolver.Address) bool {
	for i := range l {
		if l[i].Addr == addr.Addr {
			return true
		}
	}

	return false
}

func Remove(s []resolver.Address, addr resolver.Address) ([]resolver.Address, bool) {
	for i := range s {
		if s[i].Addr == addr.Addr {
			s[i] = s[len(s)-1]
			return s[:len(s)-1], true
		}
	}

	return nil, false
}
