# nanoservice/protobuf-go was based on go 1.4.2 at the time of
# writing this Dockerfile, which is why we're downloading 
# protobuf ourselves.
FROM golang
RUN apt-get update && apt-get install net-tools unzip
RUN wget https://github.com/protocolbuffers/protobuf/releases/download/v3.19.4/protoc-3.19.4-linux-x86_64.zip -O protoc.zip
RUN unzip protoc.zip -d protoc/
RUN cp protoc/bin/protoc /usr/local/bin/protoc
RUN mkdir /usr/include/google && cp -r protoc/include/google/protobuf /usr/include/google
RUN apt-get -y remove --purge unzip && apt-get -y autoremove
ADD . /go/src/github.com/otm/limes
WORKDIR /go/src/github.com/otm/limes
RUN go install google.golang.org/protobuf/proto google.golang.org/protobuf/cmd/protoc-gen-go google.golang.org/grpc/cmd/protoc-gen-go-grpc
RUN go clean
RUN go generate
RUN go get
RUN go build
RUN mkdir /root/.limes
