from golang
RUN apt-get update
RUN apt-get install net-tools
ADD . /go/src/github.com/otm/ims
WORKDIR /go/src/github.com/otm/ims
RUN go get
RUN go build

#CMD ifconfig eth0:0 169.254.169.254 netmask 255.255.255.255 && /go/src/github.com/otm/ims/ims start --config=/mnt/ims/ims.conf
#CMD ["/go/src/github.com/otm/ims/ims", "start", "--config=/mnt/ims/ims.conf"]
CMD ["./ims", "status"]
