package main

import (
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"

	pb "github.com/otm/ims/proto"
	"golang.org/x/net/context"
)

// CliHandler process calls from the cli tool
type CliHandler struct {
	address      string
	stop         chan struct{}
	log          Logger
	credsManager *CredentialsExpirationManager
}

// NewCliHandler returns a cliHandler
func NewCliHandler(address string, credsManager *CredentialsExpirationManager, stop chan struct{}) *CliHandler {
	fmt.Println("new cli handler")
	return &CliHandler{address: address, log: &ConsoleLogger{}, stop: stop, credsManager: credsManager}
}

// Start handles the cli start command
func (h *CliHandler) Start() error {
	// setupt socket
	localSocket, err := net.Listen("unix", h.address)
	if err != nil {
		return err
	}

	// we run as root, so let others connect to the socket
	h.log.Debug("Setting filemode on %v\n", h.address)
	err = os.Chmod(h.address, 0777)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	pb.RegisterInstanceMetaServiceServer(s, h)
	go s.Serve(localSocket)

	return nil
}

// Status handles the cli status command
func (h *CliHandler) Status(ctx context.Context, in *pb.Void) (*pb.StatusReply, error) {
	creds := h.credsManager.GetCredentials()

	return &pb.StatusReply{
		Error:           "",
		Role:            h.credsManager.role,
		AccessKeyId:     *creds.AccessKeyId,
		SecretAccessKey: *creds.SecretAccessKey,
		SessionToken:    *creds.SessionToken,
		Expiration:      creds.Expiration.String(),
	}, nil
}

// Stop handles the cli stop command
func (h *CliHandler) Stop(ctx context.Context, in *pb.Void) (*pb.StopReply, error) {
	close(h.stop)

	return &pb.StopReply{
		Error: "Shutting down",
	}, nil
}

// AssumeRole will switch the current role of the metadata service
func (h *CliHandler) AssumeRole(ctx context.Context, in *pb.AssumeRoleRequest) (*pb.StatusReply, error) {
	err := h.credsManager.AssumeRole(in.Name, in.Mfa)
	if err != nil {
		return nil, err
	}

	creds := h.credsManager.GetCredentials()

	return &pb.StatusReply{
		Error:           "",
		Role:            h.credsManager.role,
		AccessKeyId:     *creds.AccessKeyId,
		SecretAccessKey: *creds.SecretAccessKey,
		SessionToken:    *creds.SessionToken,
		Expiration:      creds.Expiration.String(),
	}, nil
}

// RetrieveRole assumes a role, but does not update the server
func (h *CliHandler) RetrieveRole(ctx context.Context, in *pb.AssumeRoleRequest) (*pb.StatusReply, error) {
	creds, err := h.credsManager.RetrieveRole(in.Name, in.Mfa)
	if err != nil {
		return nil, err
	}

	return &pb.StatusReply{
		Error:           "",
		Role:            h.credsManager.role,
		AccessKeyId:     *creds.AccessKeyId,
		SecretAccessKey: *creds.SecretAccessKey,
		SessionToken:    *creds.SessionToken,
		Expiration:      creds.Expiration.String(),
	}, nil
}
