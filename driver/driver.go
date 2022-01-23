package driver

import (
	"time"

	"github.com/sirupsen/logrus"
)

const (
	DefaultDriverName = "block.csi.fortress.com"
	DriverVersion     = "0.1"
	defaultTimeout    = 1 * time.Minute
)

type FortressDriver struct {
	name            string
	endpoint        string
	nodeID          string
	region          string
	publishVolumeID string
	token           string
	mountID         string
	isController    bool
	waitTimeout     time.Duration
	log             *logrus.Entry
	mounter         Mounter
	version         string
}

func NewDriver(endpoint, token, driverName, version, userAgent string) (*FortressDriver, error) {

	if driverName == "" {
		driverName = DefaultDriverName
	}

	log := logrus.New().WithFields(logrus.Fields{
		"version": version,
	})

	return &FortressDriver{
		name:         driverName,
		endpoint:     endpoint,
		isController: token != "",
		waitTimeout:  defaultTimeout,
		log:          log,
		mounter:      NewMounter(log),
		token:        token,
		version:      version,
	}, nil
}

func (d *FortressDriver) Run() {
	server := NewNonBlockingGRPCServer()
	identityServer := NewIdentityServer(d)
	nodeServer := NewNodeServer(d)
	controllerServer := NewControllerServer(d)

	server.Start(d.endpoint, identityServer, controllerServer, nodeServer)
	server.Wait()
}
