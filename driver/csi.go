package driver

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Defines Non blocking GRPC server interfaces
type NonBlockingGRPCServer interface {
	// Start services at the endpoint
	Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer)
	// Waits for the service to stop
	Wait()
	// Stops the service gracefully
	Stop()
	// Stops the service forcefully
	ForceStop()
}

type nonBlockingGRPCServer struct {
	wg     sync.WaitGroup
	server *grpc.Server
}

func NewNonBlockingGRPCServer() *nonBlockingGRPCServer {
	return &nonBlockingGRPCServer{}
}

func (s *nonBlockingGRPCServer) Wait() {
	s.wg.Wait()
}

func (s *nonBlockingGRPCServer) Stop() {
	s.server.GracefulStop()
}

func (s *nonBlockingGRPCServer) ForceStop() {
	s.server.Stop()
}

func (s *nonBlockingGRPCServer) Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) {
	s.wg.Add(1)
	go s.serve(endpoint, ids, cs, ns)
}

func (s *nonBlockingGRPCServer) serve(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) {

	proto, addr, err := parseEndpoint(endpoint)
	if err != nil {
		logrus.Fatal(err.Error())
	}

	if proto == "unix" {
		addr = "/" + addr
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			logrus.Fatalf("Failed to remove %s, error: %s", addr, err.Error())
		}
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		logrus.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logGRPC),
	}
	server := grpc.NewServer(opts...)
	s.server = server

	if ids != nil {
		csi.RegisterIdentityServer(server, ids)
	}
	if cs != nil {
		csi.RegisterControllerServer(server, cs)
	}
	if ns != nil {
		csi.RegisterNodeServer(server, ns)
	}

	logrus.WithFields(logrus.Fields{
		"proto":   proto,
		"address": addr,
	}).Infof("Listening for connections on address: %#v", listener.Addr())

	if err := server.Serve(listener); err != nil {
		log.Println(err)
	}

	s.wg.Done()

}

func parseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
}

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log := logrus.StandardLogger()
	logLevel := logrus.InfoLevel

	cut := strings.LastIndex(info.FullMethod, "/") + 1
	method := info.FullMethod[cut:]
	switch method {
	case "NodeGetCapabilities", "NodeGetVolumeStats":
		logLevel = logrus.TraceLevel
	default:
		logLevel = logrus.InfoLevel
	}

	log.Logf(logLevel, "%s: req: %+v", method, protosanitizer.StripSecrets(req))
	resp, err := handler(ctx, req)
	if err != nil {
		if logLevel == logrus.TraceLevel {
			log.Errorf("%s: req: %+v err: %v", method, protosanitizer.StripSecrets(req), err)
		} else {
			log.Errorf("%s: err: %v", method, err)
		}
	} else {
		log.Logf(logLevel, "%s: rsp: %+v", method, protosanitizer.StripSecrets(resp))
	}
	return resp, err
}
