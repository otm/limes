package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

// Common errors for credential manager
var (
	ErrMissingProfileDefault = fmt.Errorf("missing profile: default")
)

// CredentialsExpirationManager is responsible for renewing a set of credentials
type CredentialsExpirationManager struct {
	// This is the default session and information
	defaultSession     *session.Session
	defaultCredentials *sts.Credentials
	defaultSTSClient   *sts.STS

	// config is the loaded configuration
	config Config

	// roleSessionName is used when assuming roles to keep track of the user
	roleSessionName string

	// This is the current active credentials
	credLock    sync.Mutex
	role        string
	credentials *sts.Credentials
}

// NewCredentialsExpirationManager returns a credentialsExpirationManager
// It creates a session, then it will call GetSessionToken to retrieve a pair of
// temporary credentials.
func NewCredentialsExpirationManager(conf Config, mfa string) *CredentialsExpirationManager {
	defaultProfile, ok := conf.profiles["default"]
	if !ok {
		log.Fatalf("No default profile, quitting")
	}

	cm := &CredentialsExpirationManager{
		role:            "default",
		config:          conf,
		roleSessionName: defaultProfile.RoleSessionName,
	}

	sess := session.New(&aws.Config{
		Region: &defaultProfile.Region,
		Credentials: credentials.NewStaticCredentials(
			defaultProfile.AwsAccessKeyID,
			defaultProfile.AwsSecretAccessKey,
			defaultProfile.AwsSessionToken,
		),
	})
	client := sts.New(sess)

	if defaultProfile.MFASerial != "" && mfa == "" {
		log.Fatalf("MFA needed")
	}

	params := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int64(10 * 3600),
	}

	if defaultProfile.MFASerial != "" {
		params.SerialNumber = aws.String(defaultProfile.MFASerial)
	}
	if mfa != "" {
		params.TokenCode = aws.String(mfa)
	}

	resp, err := client.GetSessionToken(params)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cm.credentials = resp.Credentials
	cm.defaultCredentials = resp.Credentials
	cm.defaultSession = session.New(&aws.Config{
		Region: &defaultProfile.Region,
		Credentials: credentials.NewStaticCredentials(
			*cm.credentials.AccessKeyId,
			*cm.credentials.SecretAccessKey,
			*cm.credentials.SessionToken,
		),
	})

	cm.defaultSTSClient = sts.New(cm.defaultSession)
	go cm.Refresher()
	return cm
}

// Refresher starts a Go routine and refreshes the credentials
func (m *CredentialsExpirationManager) Refresher() {
	for {
		select {
		case <-time.After(10 * time.Second):
			m.refreshCredentials()
		}
	}
}

// AssumeRole changes (assumes) the role `name`. An optional MFA can be passed
// to the function, if set to "" the MFA is ignored
func (m *CredentialsExpirationManager) AssumeRole(name, MFA string) error {
	profile, ok := m.config.profiles[name]
	if !ok {
		return fmt.Errorf("unknown profile: %v", name)
	}

	fmt.Println("Assuming: ", name)
	return m.AssumeRoleARN(name, profile.RoleARN, profile.MFASerial, MFA)
}

// RetrieveRole will assume and fetch temporary credentials, but does not update
// the role and credentials stored by the manager.
func (m *CredentialsExpirationManager) RetrieveRole(name, MFA string) (*sts.Credentials, error) {
	profile, ok := m.config.profiles[name]
	if !ok {
		return nil, fmt.Errorf("unknown profile: %v", name)
	}

	return m.RetrieveRoleARN(profile.RoleARN, profile.MFASerial, MFA)
}

// RetrieveRoleARN assumes and fetch temporary credentials based on the RoleArn
func (m *CredentialsExpirationManager) RetrieveRoleARN(RoleARN, MFASerial, MFA string) (*sts.Credentials, error) {
	// If the default profile is requested do return the default credentials
	if m.config.profiles["default"].RoleARN == RoleARN {
		return m.defaultCredentials, nil
	}

	if MFASerial != "" && MFA == "" {
		return nil, fmt.Errorf("MFA required")
	}

	ari := &sts.AssumeRoleInput{
		DurationSeconds: aws.Int64(1800),
		RoleArn:         &RoleARN,
		RoleSessionName: &m.roleSessionName,
	}

	if MFASerial != "" {
		ari.SerialNumber = &MFASerial
	}

	if MFA != "" {
		ari.TokenCode = &MFA
	}

	resp, err := m.defaultSTSClient.AssumeRole(ari)
	if err != nil {
		return nil, err
	}

	return resp.Credentials, nil
}

// AssumeRoleARN assumes the role specified by RoleARN and will store it as
// with the name specified.
func (m *CredentialsExpirationManager) AssumeRoleARN(name, RoleARN, MFASerial, MFA string) error {
	creds, err := m.RetrieveRoleARN(RoleARN, MFASerial, MFA)
	if err != nil {
		return err
	}

	m.setCredentials(creds, name)
	return nil
}

// SetCredentials updates the stored credentials and the name of the role associated
// with the credentials
func (m *CredentialsExpirationManager) setCredentials(newCreds *sts.Credentials, role string) {
	m.credLock.Lock()
	defer m.credLock.Unlock()

	m.credentials = newCreds
	m.role = role
}

// GetCredentials returns the current saved credentials. The returned credentials
// are copied before they are returned.
func (m *CredentialsExpirationManager) GetCredentials() *sts.Credentials {
	m.credLock.Lock()
	defer m.credLock.Unlock()

	return &sts.Credentials{
		AccessKeyId:     aws.String(*m.credentials.AccessKeyId),
		Expiration:      aws.Time(*m.credentials.Expiration),
		SecretAccessKey: aws.String(*m.credentials.SecretAccessKey),
		SessionToken:    aws.String(*m.credentials.SessionToken),
	}
}

func (m *CredentialsExpirationManager) refreshCredentials() error {
	if m.defaultSTSClient == nil {
		fmt.Println("Skipping refresh: error: no default STS client")
		return errors.New("No client set for refreshing credentials")
	}

	creds := m.GetCredentials()

	if time.Now().Add(600 * time.Second).Before(*creds.Expiration) {
		// We no not need to refresh
		log.Println("Skipping refresh of credentials: ", creds.Expiration)
		return nil
	}

	if m.role == "" || m.role == "default" {
		log.Println("Skipping refresh of credentials: default role")
		// Do not refresh main default role, let it time out
		return nil
	}

	fmt.Println("====> refreshing credentials!!")
	return m.AssumeRole(m.role, "")
}
