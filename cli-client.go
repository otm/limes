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
	pb "github.com/otm/ims/proto"
	"google.golang.org/grpc"
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
func StartService(args *Start) {
	log := &ConsoleLogger{}
	config := Config{}

	// TODO: Move to function and use a default configuration file
	if args.ConfigFile != "" {
		// Parse in options from the given config file.
		log.Debug("Loading configuration from %s\n", args.ConfigFile)
		configContents, configErr := ioutil.ReadFile(args.ConfigFile)
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
		os.Remove(args.Adress)
	}()

	// Startup the HTTP server and respond to requests.
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("169.254.169.254"),
		Port: 80,
	})
	if err != nil {
		log.Fatalf("Could not startup the metadata interface: %s\n", err)
	}

	if args.MFA == "" {
		args.MFA = checkMFA(config)
	}

	credsManager := NewCredentialsExpirationManager(config, args.MFA)

	mds, metadataError := NewMetadataService(listener, credsManager)
	if metadataError != nil {
		log.Fatalf("Could not create metadata service: %s\n", metadataError.Error())
	}
	mds.Start()

	stop := make(chan struct{})
	agentServer := NewCliHandler(args.Adress, credsManager, stop, config)
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
	status := "up"

	r, err := c.srv.Status(context.Background(), &pb.Void{})
	if err != nil {
		r = &pb.StatusReply{
			Role:            "n/a",
			AccessKeyId:     "n/a",
			SecretAccessKey: "n/a",
			SessionToken:    "n/a",
			Expiration:      "n/a",
		}
		status = "down"
		defer fmt.Fprintf(os.Stderr, "\ncommunication error: %v\n", err)
	}

	if r.Error != "" {
		status = "error"
		defer fmt.Fprintf(os.Stderr, "\nerror retrieving status: %v\n", r.Error)
	}

	env := "ok"
	errConf := checkActiveAWSConfig()
	if errConf != nil {
		env = "nok"
		defer fmt.Fprintf(os.Stderr, "\nwarning: %v\n", errConf)
	}

	fmt.Printf("Server:          %v\n", status)
	fmt.Printf("AWS Config:      %v\n", env)
	fmt.Printf("Role:            %v\n", r.Role)

	if args.Verbose == false {
		return err
	}

	fmt.Printf("AccessKeyId:     %v\n", r.AccessKeyId)
	fmt.Printf("SecretAccessKey: %v\n", r.SecretAccessKey)
	fmt.Printf("SessionToken:    %v\n", r.SessionToken)
	fmt.Printf("Expiration:      %v\n", r.Expiration)

	return err
}

func (c *cliClient) assumeRole(role string, args *SwitchProfile) error {
	r, err := c.srv.AssumeRole(context.Background(), &pb.AssumeRoleRequest{Name: role})
	if err != nil {
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
