package etcd

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"os"

	"github.com/coreos/go-etcd/etcd"
	"github.com/gliderlabs/registrator/bridge"
)

func init() {
	bridge.Register(new(Factory), "etcd-tls")
}

type Factory struct{}

func (f *Factory) New(uri *url.URL) bridge.RegistryAdapter {
	urls := make([]string, 0)

	if uri.Host != "" {
		urls = append(urls, "https://"+uri.Host)
	} else {
		urls = append(urls, "https://127.0.0.1:2379")
	}

	tlskey := os.Getenv("ETCD_TLSKEY")
	tlspem := os.Getenv("ETCD_TLSPEM")
	cacert := os.Getenv("ETCD_CACERT")

  client, err := etcd.NewTLSClient(urls, tlspem, tlskey, cacert)

  if err != nil {
  	return nil
  }

	return &EtcdAdapter{client2: client, path: uri.Path}
}

type EtcdAdapter struct {
	client2 *etcd.Client

	path string
}

func (r *EtcdAdapter) Ping() error {
	r.syncEtcdCluster()

	var err error
	rr := etcd.NewRawRequest("GET", "version", nil, nil)
	_, err = r.client2.SendRequest(rr)

	if err != nil {
		return err
	}
	return nil
}

func (r *EtcdAdapter) syncEtcdCluster() {
	var result bool
	result = r.client2.SyncCluster()

	if !result {
		log.Println("etcd: sync cluster was unsuccessful")
	}
}

func (r *EtcdAdapter) Register(service *bridge.Service) error {
	r.syncEtcdCluster()

	path := r.path + "/" + service.Name + "/" + service.ID
	port := strconv.Itoa(service.Port)
	addr := net.JoinHostPort(service.IP, port)

	var err error
	_, err = r.client2.Set(path, addr, uint64(service.TTL))

	if err != nil {
		log.Println("etcd: failed to register service:", err)
	}
	return err
}

func (r *EtcdAdapter) Deregister(service *bridge.Service) error {
	r.syncEtcdCluster()

	path := r.path + "/" + service.Name + "/" + service.ID

	var err error
	_, err = r.client2.Delete(path, false)

	if err != nil {
		log.Println("etcd: failed to deregister service:", err)
	}
	return err
}

func (r *EtcdAdapter) Refresh(service *bridge.Service) error {
	return r.Register(service)
}

func (r *EtcdAdapter) Services() ([]*bridge.Service, error) {
	return []*bridge.Service{}, nil
}
