# Limes
Limes provides an easy work flow with MFA protected access keys, temporary credentials and access to multiple roles/accounts.

Limes is a Local Instance MEtadata Service and emulates parts of the [AWS Instance Metadata Service](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html) running on Amazon Linux. The AWS SDK and AWS CLI can therefor utilize this service to authenticate.

##  Installation
To be done

## Configuring the Loop Back Device
The configuration below adds the necessary IP configuration on the loop back device. Without this configuration the service can not start.

**Note:** This configuration is not persistent.

#### Linux
```
sudo ip addr add 169.254.169.254/24 broadcast 169.254.169.255 dev lo:metadata
sudo ip link set dev lo:metadata up
```

#### Mac
```
sudo /sbin/ifconfig lo0 alias 169.254.169.254
```

## Configuring IAM (Identity and Access Management)
To be done

## Configuring IMS (Instance Meta-data Service)
There is an [example configuration file](http://github.com/otm/limes/limes.conf.example). The configuration file is documented. Make a copy of the file and place it in `~/.limes/config`.

```
mkdir ~/.limes
cd ~/.limes
wget https://raw.githubusercontent.com/otm/ims/master/limes.conf.example
mv limes.conf.example config
```

Use your favorite text editor to update the configuration file



## Security
The service should be configured on the loop back device, and only accessible from the host it is running on.

**Note:** It is important not to run any service that could forwards request on the host running Limes as this would be a security risk. However, this is no difference from the setup on an Amazon Linux instance in AWS. If an attacker could forward requests to 169.254.169.254/24 your credentials could be compromised. Please note that an attacker could utilize a DNS to resolve to this address, so always be aware where you forward requests to.  

## Roadmap
* Add support for running commands

## Build
To build you need a Go compiler and environment setup. See https://golang.org/ for more information regarding setting up and configuring Go.

```
go get github.com/otm/limes
go build
```

If protobuf definitions are updated run:

```
go generate
```
