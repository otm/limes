package main

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
)

// FakeCredentialsManager will not communicate with AWS
type FakeCredentialsManager struct{}

// Role returns a dummy role name
func (m *FakeCredentialsManager) Role() string {
	return "dummy-role"
}

// Region returns the configured region for the profile or empty string if not
// defined
func (m *FakeCredentialsManager) Region() string {
	return "eu-foo-1"
}

// AssumeRole does nothing
func (m *FakeCredentialsManager) AssumeRole(name, mfa string) error {
	return nil
}

// SetSourceProfile does nothing
func (m *FakeCredentialsManager) SetSourceProfile(name, mfa string) error {
	return nil
}

// RetrieveRole return a dummy role
func (m *FakeCredentialsManager) RetrieveRole(name, MFA string) (*AwsCredentials, error) {
	c, _ := m.GetCredentials()
	return &AwsCredentials{
		Credentials: *c,
		Region:      m.Region(),
	}, nil
}

// RetrieveRoleARN returns dummy role
func (m *FakeCredentialsManager) RetrieveRoleARN(RoleARN, MFASerial, MFA string) (*sts.Credentials, error) {
	return m.GetCredentials()
}

// AssumeRoleARN does nothign
func (m *FakeCredentialsManager) AssumeRoleARN(name, RoleARN, MFASerial, MFA string) error {
	return nil
}

// GetCredentials returns fake credentials
func (m *FakeCredentialsManager) GetCredentials() (*sts.Credentials, error) {
	return &sts.Credentials{
		AccessKeyId:     aws.String("xxxxxxxxxxxx"),
		Expiration:      aws.Time(time.Now().Add(time.Duration(60 * time.Minute))),
		SecretAccessKey: aws.String("yyyyyyyyyyyyyyyyyyyyyyy"),
		SessionToken:    aws.String("xxxxxxxxxxx-yyyyyyyyyyy-zzzzzzzzzzzz"),
	}, nil
}
