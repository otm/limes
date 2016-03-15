package main

import (
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "github.com/otm/limes/proto"
	"golang.org/x/net/context"
)

// CliHandler process calls from the cli tool
type CliHandler struct {
	address      string
	stop         chan struct{}
	log          Logger
	config       Config
	credsManager CredentialsManager
}

// NewCliHandler returns a cliHandler
func NewCliHandler(address string, credsManager CredentialsManager, stop chan struct{}, config Config) *CliHandler {
	return &CliHandler{
		address:      address,
		log:          &ConsoleLogger{},
		stop:         stop,
		credsManager: credsManager,
		config:       config,
	}
}

// Start handles the cli start command
func (h *CliHandler) Start() error {
	// setupt socket
	h.log.Debug("Creating socket: %v\n", h.address)
	localSocket, err := net.Listen("unix", h.address)
	if err != nil {
		return err
	}

	// we run as root, so let others connect to the socket
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
	creds, err := h.credsManager.GetCredentials()
	if err != nil {
		if err == errMFANeeded || err == errUnknownProfile {
			return nil, grpc.Errorf(codes.FailedPrecondition, err.Error())
		}
		return nil, err
	}

	return &pb.StatusReply{
		Error:           "",
		Role:            h.credsManager.Role(),
		AccessKeyId:     *creds.AccessKeyId,
		SecretAccessKey: *creds.SecretAccessKey,
		SessionToken:    *creds.SessionToken,
		Expiration:      creds.Expiration.String(),
	}, nil
}

// Stop handles the cli stop command
func (h *CliHandler) Stop(ctx context.Context, in *pb.Void) (*pb.StopReply, error) {
	close(h.stop)

	return &pb.StopReply{}, nil
}

// AssumeRole will switch the current role of the metadata service
func (h *CliHandler) AssumeRole(ctx context.Context, in *pb.AssumeRoleRequest) (*pb.StatusReply, error) {
	err := h.credsManager.AssumeRole(in.Name, in.Mfa)
	if err != nil {
		if err == errMFANeeded || err == errUnknownProfile {
			return nil, grpc.Errorf(codes.FailedPrecondition, err.Error())
		}
		return nil, err
	}

	creds, err := h.credsManager.GetCredentials()
	if err != nil {
		if err == errMFANeeded || err == errUnknownProfile {
			return nil, grpc.Errorf(codes.FailedPrecondition, err.Error())
		}
		return nil, err
	}

	return &pb.StatusReply{
		Error:           "",
		Role:            h.credsManager.Role(),
		AccessKeyId:     *creds.AccessKeyId,
		SecretAccessKey: *creds.SecretAccessKey,
		SessionToken:    *creds.SessionToken,
		Expiration:      creds.Expiration.String(),
	}, nil
}

// AssumeRole will switch the current role of the metadata service
func (h *CliHandler) SetCredentials(ctx context.Context, in *pb.AssumeRoleRequest) (*pb.StatusReply, error) {
	err := h.credsManager.SetSourceProfile(in.Name, in.Mfa)
	if err != nil {
		if err == errMFANeeded || err == errUnknownProfile {
			return nil, grpc.Errorf(codes.FailedPrecondition, err.Error())
		}
		return nil, err
	}

	creds, err := h.credsManager.GetCredentials()
	if err != nil {
		if err == errMFANeeded || err == errUnknownProfile {
			return nil, grpc.Errorf(codes.FailedPrecondition, err.Error())
		}
		return nil, err
	}

	return &pb.StatusReply{
		Error:           "",
		Role:            h.credsManager.Role(),
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
		if err == errMFANeeded || err == errUnknownProfile {
			return nil, grpc.Errorf(codes.FailedPrecondition, err.Error())
		}
		return nil, err
	}

	return &pb.StatusReply{
		Error:           "",
		Role:            h.credsManager.Role(),
		AccessKeyId:     *creds.AccessKeyId,
		SecretAccessKey: *creds.SecretAccessKey,
		SessionToken:    *creds.SessionToken,
		Expiration:      creds.Expiration.String(),
	}, nil
}

// Config returns the current configuration
func (h *CliHandler) Config(ctx context.Context, in *pb.Void) (*pb.ConfigReply, error) {
	res := &pb.ConfigReply{
		Profiles: make(map[string]*pb.Profile, len(h.config.profiles)),
	}
	for name, profile := range h.config.profiles {
		res.Profiles[name] = &pb.Profile{
			AwsAccessKeyID:     profile.AwsAccessKeyID,
			AwsSecretAccessKey: profile.AwsSecretAccessKey,
			AwsSessionToken:    profile.AwsSessionToken,
			Region:             profile.Region,
			MFASerial:          profile.MFASerial,
			RoleARN:            profile.RoleARN,
			RoleSessionName:    profile.RoleSessionName,
		}
	}
	return res, nil
}
