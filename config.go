package main

// Config hold configuration read from the configuration file
type Config struct {
	profiles Profiles
}

// NewConfig returns a new Config struct
func NewConfig() *Config {
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
