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

	pb "github.com/otm/ims/proto"
	"google.golang.org/grpc"
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

	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithDialer(dialer))
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

	// TODO: Move to function and use a default configuration file
	if args.ConfigFile != "" {
		// Parse in options from the given config file.
		log.Debug("Loading configuration from %s\n", args.ConfigFile)
		configContents, configErr := ioutil.ReadFile(args.ConfigFile)
		if configErr != nil {
			log.Fatalf("Could not read from config file. The error was: %s\n", configErr.Error())
		}

		configParseErr := yaml.Unmarshal(configContents, config.profiles)
		if configParseErr != nil {
			log.Fatalf("Error in parsing config file: %s\n", configParseErr.Error())
		}
	} else {
		log.Debug("No configuration file given\n")
	}

	defer func() {
		log.Debug("Removing UNIX socket.\n")
		os.Remove("./ims.sock")
	}()

	// Startup the HTTP server and respond to requests.
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("169.254.169.254"),
		Port: 80,
	})
	if err != nil {
		log.Fatalf("Could not startup the metadata interface: %s\n", err)
	}

	credsManager := NewCredentialsExpirationManager(config, args.MFA)

	mds, metadataError := NewMetadataService(listener, credsManager)
	if metadataError != nil {
		log.Fatalf("Could not create metadata service: %s\n", metadataError.Error())
	}
	mds.Start()

	stop := make(chan struct{})
	agentServer := NewCliHandler("./ims.sock", credsManager, stop)
	err = agentServer.Start()
	if err != nil {
		log.Fatalf("Could not start agentServer: %s\n", err.Error())
	}

	// Wait for a graceful shutdown signal
	terminate := make(chan os.Signal)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)

	// SIGUSR1 and SIGUSR2 for controlling log level
	// respectively.
	debugEnable := make(chan os.Signal)
	debugDisable := make(chan os.Signal)
	signal.Notify(debugEnable, syscall.SIGUSR1)
	signal.Notify(debugDisable, syscall.SIGUSR2)

	log.Info("Instance Metadata Service is online, waiting for termination.\n")
	for {
		select {
		case <-stop:
			return
		case <-terminate:
			return
		case <-debugEnable:
			log.Info("Enabling debug mode.\n")
			log.Level(Debug)
		case <-debugDisable:
			log.Info("Disabling debug mode.\n")
			log.Level(Info)
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
	r, err := c.srv.Status(context.Background(), &pb.Void{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "communication error: %v\n", err)
		return err
	}

	if r.Error != "" {
		fmt.Fprintf(os.Stderr, "error stopping server: %v\n", r.Error)
		return err
	}

	fmt.Printf("Role:            %v\n", r.Role)
	fmt.Printf("AccessKeyId:     %v\n", r.AccessKeyId)
	fmt.Printf("SecretAccessKey: %v\n", r.SecretAccessKey)
	fmt.Printf("SessionToken:    %v\n", r.SessionToken)
	fmt.Printf("Expiration:      %v\n", r.Expiration)

	return nil
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
