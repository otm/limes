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
func StartService(configFile, adress, profileName, MFA string, port int, fake bool) {
	log := &ConsoleLogger{}
	config := Config{}

	// TODO: Move to function and use a default configuration file
	if configFile != "" {
		// Parse in options from the given config file.
		log.Debug("Loading configuration from %s\n", configFile)
		configContents, configErr := ioutil.ReadFile(configFile)
		if configErr != nil {
			log.Fatalf("Could not read from config file. The error was: %s\n", configErr.Error())
		}

		configParseErr := yaml.Unmarshal(configContents, &config.profiles)
		if configParseErr != nil {
			log.Fatalf("Error in parsing config file: %s\n", configParseErr.Error())
		}
	} else {
		log.Debug("No configuration file given\n")
	}

	defer func() {
		log.Debug("Removing UNIX socket.\n")
		os.Remove(adress)
	}()

	// Startup the HTTP server and respond to requests.
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("169.254.169.254"),
		Port: port,
	})
	if err != nil {
		log.Fatalf("Could not startup the metadata interface: %s\n", err)
	}

	if MFA == "" {
		MFA = checkMFA(config)
	}

	var credsManager CredentialsManager
	if fake {
		credsManager = &FakeCredentialsManager{}
	} else {
		credsManager = NewCredentialsExpirationManager(profileName, config, MFA)
	}

	mds, metadataError := NewMetadataService(listener, credsManager)
	if metadataError != nil {
		log.Fatalf("Could not create metadata service: %s\n", metadataError.Error())
	}
	mds.Start()

	stop := make(chan struct{})
	agentServer := NewCliHandler(adress, credsManager, stop, config)
	err = agentServer.Start()
	if err != nil {
		log.Fatalf("Could not start agentServer: %s\n", err.Error())
	}

	// Wait for a graceful shutdown signal
	terminate := make(chan os.Signal)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Instance Metadata Service is online, waiting for termination.\n")
	defer log.Info("Caught signal; shutting down.\n")

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
	r, err := c.srv.Stop(context.Background(), &pb.Void{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "communication error: %v\n", err)
		os.Exit(1)
	}

	if r.Error != "" {
		fmt.Fprintf(os.Stderr, "error stopping server: %v\n", r.Error)
		os.Exit(1)
	}

	fmt.Println("IMS service is stopping")

	return nil
}

func (c *cliClient) status(args *Status) error {
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

		showCorrectionAndExit(err)
		defer fmt.Fprintf(errout, "\nerror communicating with daemon: %v\n", err)
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
		defer fmt.Fprintf(errout, "run 'limes fix' to automaticly resolv the problem\n")
		defer fmt.Fprintf(errout, "\nwarning: %v\n", errConf)
	}

	if !status {
		fmt.Fprintf(out, "Status:          %v\n", "nok")
	} else {
		fmt.Fprintf(out, "Status:          %v\n", "ok")
	}
	fmt.Fprintf(out, "Role:            %v\n", r.Role)

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

func (c *cliClient) setSourceProfile(role, MFA string) error {
	r, err := c.srv.SetCredentials(context.Background(), &pb.AssumeRoleRequest{Name: role, Mfa: MFA})
	if err != nil {
		showCorrectionAndExit(err)
		fmt.Fprintf(os.Stderr, "communication error: %v\n", err)
		return err
	}

	if r.Error != "" {
		fmt.Fprintf(os.Stderr, "error stopping server: %v\n", r.Error)
		return err
	}

	fmt.Printf("Assumed: %v\n", role)
	return nil
}

func (c *cliClient) retreiveRole(role string) (*credentials.Credentials, error) {
	r, err := c.srv.RetrieveRole(context.Background(), &pb.AssumeRoleRequest{Name: role})
	if err != nil {
		showCorrectionAndExit(err)
		fmt.Fprintf(os.Stderr, "communication error: %v\n", err)
		return nil, err
	}

	if r.Error != "" {
		fmt.Fprintf(os.Stderr, "error stopping server: %v\n", r.Error)
		return nil, err
	}

	creds := credentials.NewStaticCredentials(
		r.AccessKeyId,
		r.SecretAccessKey,
		r.SessionToken,
	)

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

func checkMFA(config Config) string {
	var MFA string

	defaultProfile := config.profiles["default"]
	if defaultProfile.MFASerial == "" {
		return ""
	}

	fmt.Printf("Enter MFA: ")
	_, err := fmt.Scanf("%s", &MFA)
	if err != nil {
		log.Fatalf("err: %v\n", err)
	}

	return MFA
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
	if grpc.Code(err) == codes.FailedPrecondition {
		if grpc.ErrorDesc(err) == errMFANeeded.Error() {
			fmt.Fprintf(errout, "%v: run 'limes credentials --mfa <serial>'\n", grpc.ErrorDesc(err))
			os.Exit(1)
		}

		if grpc.ErrorDesc(err) == errUnknownProfile.Error() {
			fmt.Fprintf(errout, "%v: run 'limes --profile <name> credentials [--mfa <serial>]'\n", grpc.ErrorDesc(err))
			os.Exit(1)
		}
	}
}
