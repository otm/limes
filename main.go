package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bobziuchkovski/writ"
)

var (
	out    = os.Stdout
	errout = os.Stderr
)

var version = ""
var date = ""

const (
	configFilePath   = ".limes/config"
	domainSocketPath = ".limes/socket"
)

//go:generate protoc -I proto/ proto/ims.proto --go_out=plugins=grpc:proto

// add "port" to configuration file
// rewrite .aws/config to match current profile
// add assumable/protected directive
// limes up => start web server
// limes down => stop web server
//
// These are not well suited for multi-user systems
// ipfw show
// ipfw add 100 fwd 127.0.0.1,8080 tcp from any to any 80 in
// ipfw flush
//
// iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8080
// iptables -t nat -I OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-ports 8080
// iptables -t nat --line-numbers -n -L
//
// pf on Mac
// echo "
//   rdr pass inet proto tcp from any to any port 80 -> 127.0.0.1 port 8080
//   rdr pass inet proto tcp from any to any port 443 -> 127.0.0.1 port 8443
// " | sudo pfctl -ef

// Limes defines the cli commands
type Limes struct {
	Start         Start         `command:"start" description:"Start the Instance Metadata Service"`
	Stop          Stop          `command:"stop" description:"Stop the Instance Metadata Service"`
	Status        Status        `command:"status" description:"Get current status of the service"`
	SwitchProfile SwitchProfile `command:"assume" alias:"profile" description:"Assume IAM role"`
	RunCmd        RunCmd        `command:"run" description:"Run a command with the specified profile"`
	ShowCmd       ShowCmd       `command:"show" description:"List/show information"`
	Env           Env           `command:"env" description:"Set/clear environment variables"`
	Fix           Fix           `command:"fix" description:"Fix configuration"`
	Profile       string        `option:"profile" default:"" description:"Profile to assume"`
	ConfigFile    string        `option:"c, config" default:"" description:"Configuration file"`
	Address       string        `option:"address" default:"" description:"Address to connect to"`
	Logging       bool          `flag:"verbose" description:"Enable verbose output"`
	Version       bool          `flag:"v" description:"Show version"`
}

// Start defines the "start" command cli flags and options
type Start struct {
	HelpFlag bool   `flag:"h, help" description:"Display this message and exit"`
	Fake     bool   `flag:"fake" description:"Do not connect to AWS"`
	MFA      string `option:"m, mfa" description:"MFA token to start up server"`
	Port     int    `option:"p, port" default:"80" description:"Port used by the metadata service"`
}

// Stop defines the "stop" command cli flags and options
type Stop struct {
	HelpFlag bool `flag:"h, help" description:"Display this message and exit"`
}

// Status defines the "status" command cli flags and options
type Status struct {
	HelpFlag bool `flag:"h, help" description:"Display this message and exit"`
	Verbose  bool `flag:"v, verbose" description:"enables verbose output"`
}

// Fix defines the "fix" subcommand cli flags and options
type Fix struct {
	HelpFlag bool `flag:"h, help" description:"Display this message and exit"`
	Restore  bool `flag:"restore" description:"Restores AWS configuration files"`
}

// SwitchProfile defines the "profile" command cli flags and options
type SwitchProfile struct {
	HelpFlag bool `flag:"h, help" description:"Display this message and exit"`
}

// RunCmd defines the "run" command cli flags ands options
type RunCmd struct {
	HelpFlag bool   `flag:"h, help" description:"Display this message and exit"`
	Profile  string `option:"profile" default:"" description:"Profile to assume"`
}

// ShowCmd defines the "show" command cli flags ands options
type ShowCmd struct {
	HelpFlag bool `flag:"h, help" description:"Display this message and exit"`
}

// Env defines the "env" subcommand cli flags and options
type Env struct {
	HelpFlag bool `flag:"h, help" description:"Display this message and exit"`
	Clear    bool `flag:"clear" description:"Clear environment variables"`
}

// Run is the main cli handler
func (g *Limes) Run(cmd *Limes, p writ.Path, positional []string) {
	if g.Version {
		fmt.Printf("limes %v compiled on %v\n", version, date)
		return
	}

	p.Last().ExitHelp(errors.New("COMMAND is required"))
}

// Run is the handler for the start command
func (l *Start) Run(cmd *Limes, p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	if cmd.Profile == "" {
		cmd.Profile = "default"
	}
	StartService(cmd.ConfigFile, cmd.Address, cmd.Profile, l.MFA, l.Port, l.Fake)
}

// Run is the handler for the stop command
func (l *Stop) Run(cmd *Limes, p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	rpc := newCliClient(cmd.Address)
	defer rpc.close()
	rpc.stop(l)
}

// Run is the handler for the status command
func (l *Status) Run(cmd *Limes, p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	rpc := newCliClient(cmd.Address)
	defer rpc.close()
	rpc.status(l)
}

// Run is the handler for the fix command
func (l *Fix) Run(cmd *Limes, p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	home, err := homeDir()
	if err != nil {
		fmt.Fprintf(errout, "unable to fetch user information: %v\n", err)
		os.Exit(1)
	}

	exitOnErr := func(err error) {
		if err != nil {
			fmt.Fprintf(errout, "error: %v\n", err)
			os.Exit(1)
		}
	}

	exists := func(path string) bool {
		_, err := os.Stat(path)
		if err != nil {
			return false
		}
		return true
	}

	if l.Restore == true {
		originalPath := filepath.Join(home, awsConfDir, awsCredentialsFile)
		movedPath := filepath.Join(home, awsConfDir, awsRenamePrefix+awsCredentialsFile)
		fmt.Printf("loooking for: %v\n", movedPath)
		if exists(movedPath) {
			fmt.Fprintf(out, "# restoring: %v\n", originalPath)
			exitOnErr(os.Rename(movedPath, originalPath))
		}

		originalPath = filepath.Join(home, awsConfDir, awsConfigFile)
		movedPath = filepath.Join(home, awsConfDir, awsRenamePrefix+awsConfigFile)
		if exists(movedPath) {
			fmt.Fprintf(out, "# restoring: %v\n", originalPath)
			exitOnErr(os.Rename(movedPath, originalPath))
		}
		return
	}

	if err = checkActiveAWSConfig(); err == nil {
		fmt.Fprintf(out, "# configuration ok, nothing to fix\n")
		os.Exit(0)
	}

	for err := checkActiveAWSConfig(); err != nil; err = checkActiveAWSConfig() {
		switch err {
		case errActiveAWSCredentialsFile:
			originalPath := filepath.Join(home, awsConfDir, awsCredentialsFile)
			movedPath := filepath.Join(home, awsConfDir, awsRenamePrefix+awsCredentialsFile)
			fmt.Fprintf(out, "# moving: %v => %v\n", originalPath, movedPath)
			exitOnErr(os.Rename(originalPath, movedPath))
		case errKeyPairInAWSConfigFile:
			originalPath := filepath.Join(home, awsConfDir, awsConfigFile)
			movedPath := filepath.Join(home, awsConfDir, awsRenamePrefix+awsConfigFile)
			fmt.Fprintf(out, "# moving: %v => %v\n", originalPath, movedPath)
			exitOnErr(os.Rename(originalPath, movedPath))
		case errActiveAWSEnvironment:
			fmt.Fprintf(out, "# You have active AWS Environment variables\n")
			fmt.Fprintf(out, "# Either run the code bellow in your shell or excute it with\n")
			fmt.Fprintf(out, "# eval \"$(limes fix)\"\n")
			fmt.Fprintf(out, "unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN")
		default:
			fmt.Fprintf(out, "unable to fix: %v\n", err)
		}
	}
}

// Run is the handler for the profile command
func (l *SwitchProfile) Run(cmd *Limes, p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	if len(positional) != 1 {
		p.Last().ExitHelp(errors.New("profile name is required"))
	}

	rpc := newCliClient(cmd.Address)
	defer rpc.close()
	rpc.assumeRole(positional[0], "")
}

// Run is the handler for the run command
func (l *RunCmd) Run(cmd *Limes, p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	command := exec.Command(positional[0], positional[1:]...)

	if cmd.Profile != "" {
		rpc := newCliClient(cmd.Address)
		defer rpc.close()
		creds, err := rpc.retreiveRole(cmd.Profile, "")
		if err != nil {
			os.Exit(1)
		}
		cred, _ := creds.Get()
		command.Env = append(os.Environ(),
			"AWS_ACCESS_KEY_ID="+cred.AccessKeyID,
			"AWS_SECRET_ACCESS_KEY="+cred.SecretAccessKey,
			"AWS_SESSION_TOKEN="+cred.SessionToken,
		)
	}

	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()
	if err != nil {
		// TODO: Handle exit error
	}
}

// Run is the handler for the show subcommand
func (l *ShowCmd) Run(cmd *Limes, p writ.Path, positional []string) {
	options := []string{"profiles"}
	msg := fmt.Errorf("valid components: %v\n", strings.Join(options, ", "))

	if l.HelpFlag {
		p.Last().ExitHelp(msg)
	}

	if len(positional) == 0 {
		p.Last().ExitHelp(msg)
	}

	switch positional[0] {
	case "profiles":
		rpc := newCliClient(cmd.Address)
		defer rpc.close()
		roles, err := rpc.listRoles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%v\n", strings.Join(roles, "\n"))
	default:
		p.Last().ExitHelp(msg)
	}
}

// Run is the handler for the env subcommand
func (l *Env) Run(cmd *Limes, p writ.Path, positional []string) {
	if l.HelpFlag {
		p.Last().ExitHelp(nil)
	}

	profile := ""
	if cmd.Profile != "" {
		profile = cmd.Profile
	}

	if len(positional) == 1 {
		profile = positional[0]
	}

	if profile == "" {
		p.Last().ExitHelp(nil)
	}

	rpc := newCliClient(cmd.Address)
	defer rpc.close()

	creds, err := rpc.retreiveRole(profile, "")
	if err != nil {
		fmt.Fprintf(errout, "error retreiving profile: %v\n", err)
		os.Exit(1)
	}
	credentials, err := creds.Get()
	if err != nil {
		fmt.Fprintf(errout, "error unpacking profile: %v", err)
		os.Exit(1)
	}
	fmt.Fprintf(out, "export AWS_ACCESS_KEY_ID=%v\n", credentials.AccessKeyID)
	fmt.Fprintf(out, "export AWS_SECRET_ACCESS_KEY=%v\n", credentials.SecretAccessKey)
	fmt.Fprintf(out, "export AWS_SESSION_TOKEN=%v\n", credentials.SessionToken)
	fmt.Fprintf(out, "# Run this command to configure your shell:\n")
	fmt.Fprintf(out, "# eval \"$(limes env %s)\"\n", profile)
}

func setDefaultSocketAddress(address string) string {
	if address != "" {
		return address
	}

	home, err := homeDir()
	if err != nil {
		log.Fatalf("unable to extract user information: %v", err)
	}

	return filepath.Join(home, domainSocketPath)
}

func setDefaultConfigPath(path string) string {
	if path != "" {
		return path
	}

	home, err := homeDir()
	if err != nil {
		log.Fatalf("unable to extract user information: %v", err)
	}

	return filepath.Join(home, configFilePath)
}

func injectCmdBreak(needle string, args []string) []string {
	ret := make([]string, 0, len(args)+1)
	for _, arg := range os.Args {
		ret = append(ret, arg)
		if arg == needle {
			ret = append(ret, "--")
		}
	}
	return ret
}

func main() {
	limes := &Limes{}
	cmd := writ.New("limes", limes)
	cmd.Help.Usage = "Usage: limes [OPTIONS]... COMMAND [OPTION]... [ARG]..."
	cmd.Subcommand("start").Help.Usage = "Usage: limes start"
	cmd.Subcommand("stop").Help.Usage = "Usage: limes stop"
	cmd.Subcommand("status").Help.Usage = "Usage: limes status"
	cmd.Subcommand("fix").Help.Usage = "Usage: limes fix [--restore]"
	cmd.Subcommand("assume").Help.Usage = "Usage: limes assume <profile>"
	cmd.Subcommand("show").Help.Usage = "Usage: limes show [component]"
	cmd.Subcommand("env").Help.Usage = "Usage: limes env <profile>"
	cmd.Subcommand("run").Help.Usage = "Usage: limes [--profile <name>] run <cmd> [arg...]"

	path, positional, err := cmd.Decode(os.Args[1:])
	if err != nil {
		if path.String() != "limes run" {
			path.Last().ExitHelp(err)
		}
		os.Args = injectCmdBreak("run", os.Args)
		path, positional, err = cmd.Decode(os.Args[1:])
		if err != nil {
			path.Last().ExitHelp(err)
		}
	}

	limes.Address = setDefaultSocketAddress(limes.Address)
	limes.ConfigFile = setDefaultConfigPath(limes.ConfigFile)
	if !limes.Logging {
		log.SetOutput(ioutil.Discard)
	}

	switch path.String() {
	case "limes":
		limes.Run(limes, path, positional)
	case "limes start":
		limes.Start.Run(limes, path, positional)
	case "limes stop":
		limes.Stop.Run(limes, path, positional)
	case "limes status":
		limes.Status.Run(limes, path, positional)
	case "limes fix":
		limes.Fix.Run(limes, path, positional)
	case "limes assume":
		limes.SwitchProfile.Run(limes, path, positional)
	case "limes show":
		limes.ShowCmd.Run(limes, path, positional)
	case "limes env":
		limes.Env.Run(limes, path, positional)
	case "limes run":
		limes.RunCmd.Run(limes, path, positional)
	default:
		log.Fatalf("bug: sub command has not been setup: %v", path.String())
	}
}
