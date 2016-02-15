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
type credentialsExpirationManager struct {
	defaultSession     *session.Session
	defaultCredentials *sts.Credentials
	defaultSTSClient   *sts.STS
	config             Config
	roleSessionName    string

	credLock    sync.Mutex
	role        string
	credentials *sts.Credentials
}

// NewCredentialsExpirationManager returns a credentialsExpirationManager
// It creates a session, then it will call GetSessionToken to retrieve a pair of
// temporary credentials.
func NewCredentialsExpirationManager(conf Config, mfa string) *credentialsExpirationManager {
	defaultProfile, ok := config.profiles["default"]
	if !ok {
		log.Fatalf("No default profile, quitting")
	}

	cm := &credentialsExpirationManager{
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

	params := &sts.GetSessionTokenInput{}
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

func (m *credentialsExpirationManager) Refresher() {
	for {
		select {
		case <-time.After(10 * time.Second):
			m.maybeRefreshCredentials()
		}
	}
}

func (m *credentialsExpirationManager) AssumeRole(name, MFA string) error {
	profile, ok := m.config.profiles[name]
	if !ok {
		return fmt.Errorf("unknown profile: %v", name)
	}

	fmt.Println("Assuming: ", name)
	return m.AssumeRoleARN(name, profile.RoleARN, profile.MFASerial, MFA)
}

func (m *credentialsExpirationManager) RetrieveRole(name, MFA string) (*sts.Credentials, error) {
	profile, ok := m.config.profiles[name]
	if !ok {
		return nil, fmt.Errorf("unknown profile: %v", name)
	}

	return m.RetrieveRoleARN(name, profile.RoleARN, profile.MFASerial, MFA)
}

func (m *credentialsExpirationManager) RetrieveRoleARN(name, RoleARN, MFASerial, MFA string) (*sts.Credentials, error) {
	if name == "default" {
		return m.defaultCredentials, nil
	}

	if MFASerial != "" && MFA == "" {
		return nil, fmt.Errorf("MFA required")
	}

	ari := &sts.AssumeRoleInput{
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

func (m *credentialsExpirationManager) AssumeRoleARN(name, RoleARN, MFASerial, MFA string) error {
	creds, err := m.RetrieveRoleARN(name, RoleARN, MFASerial, MFA)
	if err != nil {
		return err
	}

	fmt.Println("setting credentials: ", creds, name)
	m.SetCredentials(creds, name)
	return nil
}

func (m *credentialsExpirationManager) SetCredentials(newCreds *sts.Credentials, role string) {
	m.credentials = newCreds
	m.role = role
}

func (m *credentialsExpirationManager) GetCredentials() (*sts.Credentials, error) {
	if m.credentials == nil {
		return nil, errors.New("No credentials set")
	}

	err := m.maybeRefreshCredentials()
	if err != nil {
		return nil, err
	}

	return m.credentials, nil
}

func (m *credentialsExpirationManager) maybeRefreshCredentials() error {
	if m.defaultSTSClient == nil {
		fmt.Println("Skipping refresh: error: no default STS client")
		return errors.New("No client set for refreshing credentials")
	}

	if time.Now().Add(600 * time.Second).Before(*m.credentials.Expiration) {
		// We no not need to refresh
		log.Println("Skipping refresh of credentials: ", m.credentials.Expiration)
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
