package main

import (
	"errors"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/bobziuchkovski/writ"
)

const (
	configFilePath   = ".limes/config"
	domainSocketPath = ".limes/socket"
)

//go:generate protoc -I proto/ proto/ims.proto --go_out=plugins=grpc:proto

// IMS defines the cli commands
type IMS struct {
	Start         Start         `command:"start" description:"Start the Instance Metadata Service"`
	Stop          Stop          `command:"stop" description:"Stop the Instance Metadata Service"`
	Status        Status        `command:"status" description:"Get current status of the service"`
	SwitchProfile SwitchProfile `command:"profile" description:"Assume IAM role"`
	//RunCmd        RunCmd        `command:"run" description:"Run a command with the specified profile"`
}

// Start defines the "start" command cli flags and options
type Start struct {
	HelpFlag   bool   `flag:"h, help" description:"START Display this message and exit"`
	MFA        string `option:"m, mfa" description:"MFA token to start up server"`
	ConfigFile string `option:"c, config" default:"" description:"configuration file"`
	Adress     string `option:"adress" default:"" description:"addess to local socket communication"`
}

// Stop defines the "stop" command cli flags and options
type Stop struct {
	HelpFlag   bool   `flag:"h, help" description:"STOP Display this message and exit"`
	ConfigFile string `option:"c, config" default:"" description:"configuration file"`
	Adress     string `option:"adress" default:"" description:"configuration file"`
}

// Status defines the "status" command cli flags and options
type Status struct {
	HelpFlag   bool   `flag:"h, help" description:"STATUS Display this message and exit"`
	ConfigFile string `option:"c, config" default:"" description:"configuration file"`
	Adress     string `option:"adress" default:"" description:"configuration file"`
	Verbose    bool   `flag:"v, verbose" description:"enables verbose output"`
}

// SwitchProfile defines the "profile" command cli flags and options
type SwitchProfile struct {
	HelpFlag   bool   `flag:"h, help" description:"SwitchProfile Display this message and exit"`
	ConfigFile string `option:"c, config" default:"" description:"configuration file"`
	Adress     string `option:"adress" default:"" description:"configuration file"`
}

// RunCmd defines the "run" command cli flags ands options
// type RunCmd struct {
// 	HelpFlag   bool   `flag:"h, help" description:"RunCmd Display this message and exit"`
// 	Profile    string `option:"p, profile" default:"" description:"profile to assume"`
// 	ConfigFile string `option:"c, config" default:"./ims.conf" description:"configuration file"`
// 	Adress     string `option:"adress" default:"./ims.sock" description:"configuration file"`
// }

// Run is the main cli handler
func (g *IMS) Run(p writ.Path, positional []string) {
	p.Last().ExitHelp(errors.New("COMMAND is required"))
}

// Run is the handler for the start command
func (l *Start) Run(p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	l.ConfigFile = setDefaultConfigPath(l.ConfigFile)
	l.Adress = setDefaultSocketAdress(l.Adress)
	StartService(l)
}

// Run is the handler for the stop command
func (l *Stop) Run(p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	l.ConfigFile = setDefaultConfigPath(l.ConfigFile)
	l.Adress = setDefaultSocketAdress(l.Adress)
	rpc := newCliClient(l.Adress)
	defer rpc.close()
	rpc.stop(l)
}

// Run is the handler for the status command
func (l *Status) Run(p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	l.ConfigFile = setDefaultConfigPath(l.ConfigFile)
	l.Adress = setDefaultSocketAdress(l.Adress)
	rpc := newCliClient(l.Adress)
	defer rpc.close()
	rpc.status(l)
}

// Run is the handler for the profile command
func (l *SwitchProfile) Run(p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	if len(positional) != 1 {
		p.Last().ExitHelp(errors.New("profile name is required"))
	}

	l.ConfigFile = setDefaultConfigPath(l.ConfigFile)
	l.Adress = setDefaultSocketAdress(l.Adress)
	rpc := newCliClient(l.Adress)
	defer rpc.close()
	rpc.assumeRole(positional[0], l)
}

// Run is the handler for the run command
// func (l *RunCmd) Run(p writ.Path, positional []string) {
// 	if l.HelpFlag {
// 		p.Last().ExitHelp(nil)
// 	}
//
// 	fmt.Printf("Run is not implemented")
// }

func setDefaultSocketAdress(adress string) string {
	if adress != "" {
		return adress
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatalf("unable to extract user information: %v", err)
	}

	return filepath.Join(usr.HomeDir, domainSocketPath)
}

func setDefaultConfigPath(path string) string {
	if path != "" {
		return path
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatalf("unable to extract user information: %v", err)
	}

	return filepath.Join(usr.HomeDir, configFilePath)
}

func main() {
	ims := &IMS{}
	cmd := writ.New("ims", ims)
	cmd.Help.Usage = "Usage: ims COMMAND [OPTION]... [ARG]..."
	cmd.Subcommand("start").Help.Usage = "Usage: ims start [--mfa <token>]"
	cmd.Subcommand("stop").Help.Usage = "Usage: ims stop"
	cmd.Subcommand("status").Help.Usage = "Usage: ims status"
	cmd.Subcommand("profile").Help.Usage = "Usage: ims profile [name]"
	//cmd.Subcommand("run").Help.Usage = "Usage: ims run [--profile <name>] <cmd> [arg...]"

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
	// case "ims run":
	// 	ims.RunCmd.Run(path, positional)
	default:
		log.Fatalf("bug: sub command has not been setup: %v", path.String())
	}

}
