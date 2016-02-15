package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/bobziuchkovski/writ"
)

//go:generate protoc -I proto/ proto/ims.proto --go_out=plugins=grpc:proto

// IMS is the base cli handler
type IMS struct {
	Start         Start         `command:"start" description:"Start the Instance Metadata Service"`
	Stop          Stop          `command:"stop" description:"Stop the Instance Metadata Service"`
	Status        Status        `command:"status" description:"Get current status of the service"`
	SwitchProfile SwitchProfile `command:"profile" description:"Assume IAM role"`
	RunCmd        RunCmd        `command:"run" description:"Run a command with the specified profile"`
}

// Start defines cli flags
type Start struct {
	HelpFlag   bool   `flag:"h, help" description:"START Display this message and exit"`
	MFA        string `option:"m, mfa" description:"MFA token to start up server"`
	ConfigFile string `option:"c, config" default:"./ims.conf" description:"configuration file"`
	Adress     string `option:"adress" default:"./ims.sock" description:"configuration file"`
}

// Stop defines cli flags
type Stop struct {
	HelpFlag   bool   `flag:"h, help" description:"STOP Display this message and exit"`
	ConfigFile string `option:"c, config" default:"./ims.conf" description:"configuration file"`
	Adress     string `option:"adress" default:"./ims.sock" description:"configuration file"`
}

// Status defines cli flags
type Status struct {
	HelpFlag   bool   `flag:"h, help" description:"STATUS Display this message and exit"`
	ConfigFile string `option:"c, config" default:"./ims.conf" description:"configuration file"`
	Adress     string `option:"adress" default:"./ims.sock" description:"configuration file"`
}

// SwitchProfile defines cli flags
type SwitchProfile struct {
	HelpFlag   bool   `flag:"h, help" description:"SwitchProfile Display this message and exit"`
	ConfigFile string `option:"c, config" default:"./ims.conf" description:"configuration file"`
	Adress     string `option:"adress" default:"./ims.sock" description:"configuration file"`
}

// RunCmd defines cli flags
type RunCmd struct {
	HelpFlag   bool   `flag:"h, help" description:"RunCmd Display this message and exit"`
	Profile    string `option:"p, profile" default:"" description:"profile to assume"`
	ConfigFile string `option:"c, config" default:"./ims.conf" description:"configuration file"`
	Adress     string `option:"adress" default:"./ims.sock" description:"configuration file"`
}

// Run is the main cli handler
func (g *IMS) Run(p writ.Path, positional []string) {
	p.Last().ExitHelp(errors.New("COMMAND is required"))
}

// Run is the handler for the start command
func (l *Start) Run(p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	StartService(l)

	// FIXME: Uniform way to create a logger
	// log.Info("Caught signal; shutting down now.\n")
}

// Run is a cli handler
func (l *Stop) Run(p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	rpc := newCliClient(l.Adress)
	defer rpc.close()
	rpc.stop(l)
}

// Run is the cli handler for status
func (l *Status) Run(p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	rpc := newCliClient(l.Adress)
	defer rpc.close()
	rpc.status(l)
}

// Run is a cli handler
func (l *SwitchProfile) Run(p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	if len(positional) != 1 {
		p.Last().ExitHelp(errors.New("profile name is required"))
	}

	rpc := newCliClient(l.Adress)
	defer rpc.close()
	rpc.assumeRole(positional[0], l)
}

// Run is a cli handler
func (l *RunCmd) Run(p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	fmt.Printf("Run is not implemented")
}

// Config hold configuration read from the configuration file
type Config struct {
	profiles Profiles
}

func newConfig() *Config {
	config := &Config{
		profiles: make(Profiles),
	}

	return config
}

// Profiles is a map for AWS profiles
type Profiles map[string]Profile

// Profile defines an AWS IAM profile
type Profile struct {
	AwsAccessKeyID     string `yaml:"aws_access_key_id"`
	AwsSecretAccessKey string `yaml:"aws_secret_access_key"`
	AwsSessionToken    string
	Region             string `yaml:"region"`
	MFASerial          string `yaml:"mfa_serial"`
	RoleARN            string `yaml:"role_arn"`
	SourceProfile      string `yaml:"source_profile"`
	RoleSessionName    string `yaml:"role_session_name"`
}

var (
	debugMode  = flag.Bool("debug", false, "Enable debug mode.")
	configFile = flag.String("conf", "", "Config file to load.")
	config     = *newConfig()
)

func main() {
	ims := &IMS{}
	cmd := writ.New("ims", ims)
	cmd.Help.Usage = "Usage: ims COMMAND [OPTION]... [ARG]..."
	cmd.Subcommand("start").Help.Usage = "Usage: ims start [--mfa <token>]"
	cmd.Subcommand("stop").Help.Usage = "Usage: ims stop"
	cmd.Subcommand("status").Help.Usage = "Usage: ims status"
	cmd.Subcommand("profile").Help.Usage = "Usage: ims profile [name]"
	cmd.Subcommand("run").Help.Usage = "Usage: ims run [--profile <name>] <cmd> [arg...]"

	path, positional, err := cmd.Decode(os.Args[1:])
	if err != nil {
		path.Last().ExitHelp(err)
	}
	switch path.String() {
	case "ims":
		ims.Run(path, positional)
	case "ims start":
		ims.Start.Run(path, positional)
	case "ims stop":
		ims.Stop.Run(path, positional)
	case "ims status":
		ims.Status.Run(path, positional)
	case "ims profile":
		ims.SwitchProfile.Run(path, positional)
	case "ims run":
		ims.RunCmd.Run(path, positional)
	default:
		panic("BUG: Someone added a new command and forgot to add it's path here")
	}

}
