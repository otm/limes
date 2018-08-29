package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/context"

	"gopkg.in/yaml.v2"

	"github.com/aws/aws-sdk-go/aws/credentials"
	pb "github.com/otm/limes/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
)

type awsEnv struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Region          string
}

type cliClient struct {
	conn *grpc.ClientConn
	srv  pb.InstanceMetaServiceClient
}

func newCliClient(address string) *cliClient {
	client := &cliClient{}

	dialer := func(addr string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout("unix", addr, timeout)
	}

	grpclog.SetLogger(log.New(ioutil.Discard, "", log.LstdFlags))

	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithDialer(dialer), grpc.WithTimeout(1*time.Second))
	if err != nil {
		log.Fatalf("did not connect: %v\n", err)
	}

	client.conn = conn
	client.srv = pb.NewInstanceMetaServiceClient(conn)

	return client
}

// StartService bootstraps the metadata service
func StartService(configFile, address, profileName, MFA string, port int, fake bool) {
	log := &ConsoleLogger{}
	config := Config{}

	// TODO: Move to function and use a default configuration file
	if configFile != "" {
		// Parse in options from the given config file.
		log.Debug("Loading configuration: %s\n", configFile)
		configContents, configErr := ioutil.ReadFile(configFile)
		if configErr != nil {
			log.Fatalf("Error reading config: %s\n", configErr.Error())
		}

		configParseErr := yaml.Unmarshal(configContents, &config)
		if configParseErr != nil {
			log.Fatalf("Error in parsing config file: %s\n", configParseErr.Error())
		}

		if len(config.Profiles) == 0 {
			log.Info("No profiles found, falling back to old config format.\n")
			configParseErr := yaml.Unmarshal(configContents, &config.Profiles)
			if configParseErr != nil {
				log.Fatalf("Error in parsing config file: %s\n", configParseErr.Error())
			}
			if len(config.Profiles) > 0 {
				log.Warning("WARNING: old deprecated config format is used.\n")
			}
		}
	} else {
		log.Debug("No configuration file given\n")
	}

	defer func() {
		log.Debug("Removing socket: %v\n", address)
		os.Remove(address)
	}()

	if port == 0 {
		port = config.Port
	}

	if port == 0 {
		port = 80
	}

	// Startup the HTTP server and respond to requests.
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("169.254.169.254"),
		Port: port,
	})
	if err != nil {
		log.Fatalf("Failed to bind to socket: %s\n", err)
	}

	var credsManager CredentialsManager
	if fake {
		credsManager = &FakeCredentialsManager{}
	} else {
		credsManager = NewCredentialsExpirationManager(profileName, config, MFA)
	}

	log.Info("Starting web service: %v:%v\n", "169.254.169.254", port)
	mds, metadataError := NewMetadataService(listener, credsManager)
	if metadataError != nil {
		log.Fatalf("Failed to start metadata service: %s\n", metadataError.Error())
	}
	mds.Start()

	stop := make(chan struct{})
	agentServer := NewCliHandler(address, credsManager, stop, config)
	err = agentServer.Start()
	if err != nil {
		log.Fatalf("Failed to start agentServer: %s\n", err.Error())
	}

	// Wait for a graceful shutdown signal
	terminate := make(chan os.Signal)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Service: online\n")
	defer log.Info("Caught signal: shutting down.\n")

	for {
		select {
		case <-stop:
			return
		case <-terminate:
			return
		}
	}
}

func (c *cliClient) close() error {
	return c.conn.Close()
}

func (c *cliClient) stop(args *Stop) error {
	_, err := c.srv.Stop(context.Background(), &pb.Void{})
	if err != nil {
		if grpc.ErrorDesc(err) == grpc.ErrClientConnClosing.Error() {
			return nil
		}

		fmt.Fprintf(os.Stderr, "limes: unknown error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Limes service is stopping")

	return nil
}

func (c *cliClient) printStatus(args *Status) error {
	status := true

	service := "up"
	r, err := c.srv.Status(context.Background(), &pb.Void{})
	if err != nil {
		r = &pb.StatusReply{
			Role:            "n/a",
			AccessKeyId:     "n/a",
			SecretAccessKey: "n/a",
			SessionToken:    "n/a",
			Expiration:      "n/a",
		}
		service = "down"
		status = false

		defer fmt.Fprintf(errout, "\n%v", lookupCorrection(err))
	}

	if r.Error != "" {
		service = "error"
		status = false
		defer fmt.Fprintf(errout, "\nerror communication with daemon: %v\n", r.Error)
	}

	env := "ok"
	errConf := checkActiveAWSConfig()
	if errConf != nil {
		env = "nok"
		status = false
		defer fmt.Fprintf(errout, "run 'limes fix' to automatically resolve the problem\n")
		defer fmt.Fprintf(errout, "\nwarning: %v\n", errConf)
	}

	if !status {
		fmt.Fprintf(out, "Status:          %v\n", "nok")
	} else {
		fmt.Fprintf(out, "Status:          %v\n", "ok")
	}
	fmt.Fprintf(out, "Profile:            %v\n", r.Role)

	if args.Verbose == false {
		return err
	}

	fmt.Fprintf(out, "Server:          %v\n", service)
	fmt.Fprintf(out, "AWS Config:      %v\n", env)
	fmt.Fprintf(out, "AccessKeyId:     %v\n", r.AccessKeyId)
	fmt.Fprintf(out, "SecretAccessKey: %v\n", r.SecretAccessKey)
	fmt.Fprintf(out, "SessionToken:    %v\n", r.SessionToken)
	fmt.Fprintf(out, "Expiration:      %v\n", r.Expiration)

	return err
}

func (c *cliClient) status() (*pb.StatusReply, error) {
	return c.srv.Status(context.Background(), &pb.Void{})
}

func (c *cliClient) assumeRole(role string, MFA string) error {
	r, err := c.srv.AssumeRole(context.Background(), &pb.AssumeRoleRequest{Name: role, Mfa: MFA})
	if err != nil {
		if grpc.Code(err) == codes.FailedPrecondition && grpc.ErrorDesc(err) == errMFANeeded.Error() {
			return c.assumeRole(role, askMFA())
		}

		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return err
	}

	fmt.Fprintf(out, "Assumed: %v\n", r.Role)
	return nil
}

func (c *cliClient) retreiveRole(role, MFA string) (*credentials.Credentials, error) {
	r, err := c.srv.RetrieveRole(context.Background(), &pb.AssumeRoleRequest{Name: role, Mfa: MFA})
	if err != nil {
		if grpc.Code(err) == codes.FailedPrecondition && grpc.ErrorDesc(err) == errMFANeeded.Error() {
			return c.retreiveRole(role, askMFA())
		}

		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return nil, err
	}

	creds := credentials.NewStaticCredentials(
		r.AccessKeyId,
		r.SecretAccessKey,
		r.SessionToken,
	)

	return creds, nil
}

func (c *cliClient) retreiveAWSEnv(role, MFA string) (awsEnv, error) {
	r, err := c.srv.RetrieveRole(context.Background(), &pb.AssumeRoleRequest{Name: role, Mfa: MFA})
	if err != nil {
		if grpc.Code(err) == codes.FailedPrecondition && grpc.ErrorDesc(err) == errMFANeeded.Error() {
			return c.retreiveAWSEnv(role, askMFA())
		}

		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return awsEnv{}, err
	}

	creds := awsEnv{
		AccessKeyID:     r.AccessKeyId,
		SecretAccessKey: r.SecretAccessKey,
		SessionToken:    r.SessionToken,
		Region:          r.Region,
	}

	return creds, nil
}

// Config(ctx context.Context, in *Void, opts ...grpc.CallOption) (*ConfigReply, error)
func (c *cliClient) listRoles() ([]string, error) {
	r, err := c.srv.Config(context.Background(), &pb.Void{})
	if err != nil {
		showCorrectionAndExit(err)
		fmt.Fprintf(os.Stderr, "communication error: %v\n", err)
		return nil, err
	}

	roles := make([]string, 0, len(r.Profiles))
	for role := range r.Profiles {
		roles = append(roles, role)
	}

	return roles, nil
}

// ask the user for an MFA token
func askMFA() string {
	var MFA string

	fmt.Printf("Enter MFA: ")
	_, err := fmt.Scanf("%s", &MFA)
	if err != nil {
		log.Fatalf("err: %v\n", err)
	}

	return MFA
}

func showCorrectionAndExit(err error) {
	fmt.Fprintf(errout, lookupCorrection(err))
	os.Exit(1)
}

func lookupCorrection(err error) string {
	switch grpc.Code(err) {
	case codes.FailedPrecondition:
		switch grpc.ErrorDesc(err) {
		case errMFANeeded.Error():
			return fmt.Sprintf("%v: run 'limes assume <profile>'\n", grpc.ErrorDesc(err))
		case errUnknownProfile.Error():
			return fmt.Sprintf("%v: run 'limes assume <profile>'\n", grpc.ErrorDesc(err))
		}
	case codes.Unknown:
		switch grpc.ErrorDesc(err) {
		case grpc.ErrClientConnClosing.Error(), grpc.ErrClientConnTimeout.Error():
			return fmt.Sprintf("service down: run 'limes start'\n")
		}
	}
	return fmt.Sprintf("%s\n", err)
}
