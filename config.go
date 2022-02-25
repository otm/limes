package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

// Config hold configuration read from the configuration file
type Config struct {
	Port    int    `yaml:"port"`
	Address string `yaml:"address"`
	Profiles
}

// NewConfig returns a new Config struct
func NewConfig() *Config {
	config := &Config{
		Profiles: make(Profiles),
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
	Protected          bool   `yaml:"protected"`
}

func (p Profile) protected() bool {
	return p.Protected
}

const (
	awsConfDir         = ".aws"
	awsConfigFile      = "config"
	awsCredentialsFile = "credentials"
	awsRenamePrefix    = "limes."
	awsAccessKeyEnv    = "AWS_ACCESS_KEY_ID"
	awsSecretKeyEnv    = "AWS_SECRET_ACCESS_KEY"
	limesConfFlag      = "generatedBy=limes"
)

var (
	errActiveAWSEnvironment        = fmt.Errorf("active AWS environment variables")
	errActiveAWSCredentialsFile    = fmt.Errorf("active AWS credentials file")
	errActiveAWSConfigFile         = fmt.Errorf("active AWS config file")
	errKeyPairInAWSConfigFile      = fmt.Errorf("active AWS key pair in config file")
	errKeyPairInAWSCredentialsFile = fmt.Errorf("active AWS key pair in credentials file")
)

func checkActiveAWSConfig() (err error) {
	var active bool

	if active, err = doCheck(activeAWSEnvironment, err); active {
		return errActiveAWSEnvironment
	}

	if active, err = doCheck(activeAWSCredentialsFile, err); active {
		if active, err = doCheck(credentialsInAWSCredentialsFile, nil); active {
			return errKeyPairInAWSConfigFile
		}
	}

	if active, err = doCheck(activeAWSConfigFile, err); active {
		if active, err = doCheck(credentialsInAWSConfigFile, nil); active {
			return errKeyPairInAWSConfigFile
		}
	}

	return err
}

func doCheck(fn func() (bool, error), err error) (bool, error) {
	if err != nil {
		return false, err
	}

	return fn()
}

func activeAWSConfigFile() (active bool, err error) {
	home, err := homeDir()
	if err != nil {
		return false, fmt.Errorf("unable to fetch user information: %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, awsConfDir, awsConfigFile)); os.IsNotExist(err) {
		return false, nil
	}

	return true, nil
}

func credentialsInAWSConfigFile() (active bool, err error) {
	home, err := homeDir()
	if err != nil {
		return false, fmt.Errorf("unable to fetch user information: %v", err)
	}

	file, err := os.Open(filepath.Join(home, awsConfDir, awsConfigFile))
	if err != nil {
		return false, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txt := scanner.Text()
		if strings.Contains(txt, "aws_access_key_id") {
			return true, errKeyPairInAWSConfigFile
		}
		if strings.Contains(txt, "aws_secret_access_key") {
			return true, errKeyPairInAWSConfigFile
		}
	}

	return false, nil
}

func credentialsInAWSCredentialsFile() (active bool, err error) {
	home, err := homeDir()
	if err != nil {
		return false, fmt.Errorf("unable to fetch user information: %v", err)
	}

	file, err := os.Open(filepath.Join(home, awsConfDir, awsCredentialsFile))
	if err != nil {
		return false, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txt := scanner.Text()
		if strings.Contains(txt, "aws_access_key_id") {
			return true, errKeyPairInAWSConfigFile
		}
		if strings.Contains(txt, "aws_secret_access_key") {
			return true, errKeyPairInAWSConfigFile
		}
	}

	return false, nil
}

func limesGeneratedConfigFile() (active bool, err error) {
	home, err := homeDir()
	if err != nil {
		return false, fmt.Errorf("unable to fetch user information: %v", err)
	}

	file, err := os.Open(filepath.Join(home, awsConfDir, awsConfigFile))
	if err != nil {
		return false, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txt := scanner.Text()
		if strings.Contains(txt, "generatedBy=limes") {
			return true, nil
		}
	}

	return false, nil
}

func limesGeneratedCredentialsFile() (active bool, err error) {
	home, err := homeDir()
	if err != nil {
		return false, fmt.Errorf("unable to fetch user information: %v", err)
	}

	file, err := os.Open(filepath.Join(home, awsConfDir, awsCredentialsFile))
	if err != nil {
		return false, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txt := scanner.Text()
		if strings.Contains(txt, "generatedBy=limes") {
			return true, nil
		}
	}

	return false, nil
}

func activeAWSCredentialsFile() (active bool, err error) {
	home, err := homeDir()
	if err != nil {
		return false, fmt.Errorf("unable to fetch user information: %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, awsConfDir, awsCredentialsFile)); os.IsNotExist(err) {
		return false, nil
	}

	return true, nil
}

func activeAWSEnvironment() (active bool, err error) {
	for _, e := range os.Environ() {
		switch strings.Split(e, "=")[0] {
		case awsAccessKeyEnv:
			return true, nil
		case awsSecretKeyEnv:
			return true, nil
		}
	}

	return false, nil
}

func homeDir() (homeDir string, err error) {
	usr, err := user.Current()
	// find real user when called with sudo
	sudoer := os.Getenv("SUDO_USER")
	if sudoer != "" {
		usr, err = user.Lookup(sudoer)
	}
	if err == nil {
		return usr.HomeDir, nil
	}

	// fallback to environemt variables
	var home string
	if runtime.GOOS == "windows" {
		home = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home != "" {
			return home, nil
		}

		home = os.Getenv("USERPROFILE")
		if home != "" {
			return home, nil
		}

		return "", fmt.Errorf("fallback failed, set `USERPROFILE` environment variable")
	}

	home = os.Getenv("HOME")
	if home != "" {
		return home, nil
	}

	return "", fmt.Errorf("fallback failed, set `HOME` environment variable")
}

func writeAwsConfig(region string) error {
	configFile := os.Getenv("AWS_CONFIG_FILE")
	if configFile == "" {
		home, err := homeDir()
		if err != nil {
			return err
		}
		configFile = filepath.Join(home, awsConfDir, awsConfigFile)
	}

	active, err := activeAWSConfigFile()
	if err != nil {
		return err
	}

	if active {
		limesGenerated, err := limesGeneratedConfigFile()
		if err != nil {
			return err
		}
		if !limesGenerated {
			return errActiveAWSConfigFile
		}
	}

	conf := []byte(fmt.Sprintf("[default]\nregion=%s\n%s", region, limesConfFlag))
	ioutil.WriteFile(configFile, conf, 0600)

	return nil
}

func writeAwsCredentials(region string) error {
	configFile := os.Getenv("AWS_CREDENTIAL_FILE")
	if configFile == "" {
		home, err := homeDir()
		if err != nil {
			return err
		}
		configFile = filepath.Join(home, awsConfDir, awsCredentialsFile)
	}

	active, err := activeAWSCredentialsFile()
	if err != nil {
		return err
	}

	if active {
		limesGenerated, err := limesGeneratedCredentialsFile()
		if err != nil {
			return err
		}
		if !limesGenerated {
			return errActiveAWSConfigFile
		}
	}

	conf := []byte(fmt.Sprintf("[default]\nregion=%s\n%s", region, limesConfFlag))
	ioutil.WriteFile(configFile, conf, 0600)

	return nil
}
