package discovery

import (
	"context"
	"time"

	"github.com/morris-zheng/go-slim-core/logger"

	clientV3 "go.etcd.io/etcd/client/v3"
)

type Register struct {
	Endpoints     []string
	Prefix        string
	TTL           int64
	client        *clientV3.Client
	logger        logger.Logger
	node          *Node
	leasesID      clientV3.LeaseID
	keepAliveChan <-chan *clientV3.LeaseKeepAliveResponse
}

func NewRegister(opt Option, l *logger.Logger) (*Register, error) {
	schemeName = opt.Prefix
	r := &Register{
		Prefix:    opt.Prefix,
		Endpoints: opt.Endpoints,
		TTL:       opt.TTL,
		logger:    *l,
	}

	client, err := clientV3.New(clientV3.Config{
		Endpoints:   opt.Endpoints,
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		return nil, err
	}

	r.client = client

	return r, nil
}

func (r *Register) Register(n *Node) error {
	r.node = n

	if err := r.register(); err != nil {
		return err
	}

	go r.keepAlive()

	return nil
}

func (r *Register) register() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := r.client.Grant(ctx, r.TTL)
	if err != nil {
		return err
	}

	r.leasesID = resp.ID

	if r.keepAliveChan, err = r.client.KeepAlive(context.Background(), r.leasesID); err != nil {
		return err
	}

	_, err = r.client.Put(context.Background(), r.node.Key(r.Prefix), r.node.Encode(), clientV3.WithLease(r.leasesID))

	return err
}

func (r *Register) Deregister() {
	_, err := r.client.Delete(context.Background(), r.node.Key(r.Prefix))
	if err != nil {
		r.logger.Error(context.Background(), "unregister failed, error: ", err)
	}

	if _, err := r.client.Revoke(context.Background(), r.leasesID); err != nil {
		r.logger.Error(context.Background(), "revoke failed, error: ", err)
	}
}

func (r *Register) keepAlive() {
	ticker := time.NewTicker(time.Duration(r.TTL) * time.Second)
	for {
		select {
		case res := <-r.keepAliveChan:
			if res == nil {
				if err := r.register(); err != nil {
					r.logger.Error(context.Background(), "register failed, error: ", err)
				}
			}
		case <-ticker.C:
			if r.keepAliveChan == nil {
				if err := r.register(); err != nil {
					r.logger.Error(context.Background(), "register failed, error: ", err)
				}
			}
		}
	}
}
