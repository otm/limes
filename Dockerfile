# nanoservice/protobuf-go was based on go 1.4.2 at the time of
# writing this Dockerfile, which is why we're downloading 
# protobuf ourselves.
from golang
RUN apt-get update && apt-get install net-tools unzip
RUN wget https://github.com/google/protobuf/releases/download/v3.0.0-beta-2/protoc-3.0.0-beta-2-linux-x86_64.zip -O protoc.zip
RUN unzip protoc.zip -d protoc/ 
RUN cp protoc/protoc /usr/local/bin/protoc 
RUN mkdir /usr/include/google && cp -r protoc/google/protobuf /usr/include/google
RUN apt-get -y remove --purge unzip && apt-get -y autoremove
RUN go get -u github.com/golang/protobuf/proto github.com/golang/protobuf/protoc-gen-go 
ADD . /go/src/github.com/otm/limes
WORKDIR /go/src/github.com/otm/limes
RUN go clean
RUN go generate
RUN go get
RUN go build
RUN mkdir /root/.limes

