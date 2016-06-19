package eureka

/*
	Eureka is a plugin for Netflix Eureka service discovery
*/

import (
	"time"

	"github.com/hudl/fargo"
	"github.com/micro/go-micro/cmd"
	"github.com/micro/go-micro/registry"
	"github.com/op/go-logging"
)

type fargoConnection interface {
	RegisterInstance(*fargo.Instance) error
	DeregisterInstance(*fargo.Instance) error
	HeartBeatInstance(*fargo.Instance) error
	GetInstance(string, string) (*fargo.Instance, error)
	GetApp(string) (*fargo.Application, error)
	GetApps() (map[string]*fargo.Application, error)
	ScheduleAppUpdates(string, bool, <-chan struct{}) <-chan fargo.AppUpdate
}

type eurekaRegistry struct {
	conn fargoConnection
	opts registry.Options
}

func init() {
	cmd.DefaultRegistries["eureka"] = NewRegistry
	logging.SetLevel(logging.ERROR, "fargo")
}

func newRegistry(opts ...registry.Option) registry.Registry {
	var options registry.Options
	for _, o := range opts {
		o(&options)
	}

	var cAddrs []string
	for _, addr := range options.Addrs {
		if len(addr) == 0 {
			continue
		}
		cAddrs = append(cAddrs, addr)
	}

	if len(cAddrs) == 0 {
		cAddrs = []string{"http://localhost:8080/eureka/v2"}
	}

	conn := fargo.NewConn(cAddrs...)
	conn.PollInterval = time.Second * 5

	return &eurekaRegistry{
		conn: &conn,
		opts: options,
	}
}

func (e *eurekaRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	instance, err := serviceToInstance(s)
	if err != nil {
		return err
	}

	if e.instanceRegistered(instance) {
		return e.conn.HeartBeatInstance(instance)
	}

	return e.conn.RegisterInstance(instance)
}

func (e *eurekaRegistry) Deregister(s *registry.Service) error {
	instance, err := serviceToInstance(s)
	if err != nil {
		return err
	}
	return e.conn.DeregisterInstance(instance)
}

func (e *eurekaRegistry) GetService(name string) ([]*registry.Service, error) {
	app, err := e.conn.GetApp(name)
	if err != nil {
		return nil, err
	}
	return appToService(app), nil
}

func (e *eurekaRegistry) ListServices() ([]*registry.Service, error) {
	var services []*registry.Service

	apps, err := e.conn.GetApps()
	if err != nil {
		return nil, err
	}

	for _, app := range apps {
		services = append(services, appToService(app)...)
	}

	return services, nil
}

func (e *eurekaRegistry) Watch() (registry.Watcher, error) {
	return newWatcher(e.conn), nil
}

func (e *eurekaRegistry) String() string {
	return "eureka"
}

func (e *eurekaRegistry) instanceRegistered(instance *fargo.Instance) bool {
	_, err := e.conn.GetInstance(instance.App, instance.UniqueID(*instance))
	return err == nil
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	return newRegistry(opts...)
}
