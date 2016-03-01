# nanoservice/protobuf-go was based on go 1.4.2 at the time of
# writing this Dockerfile, which is why we're downloading 
# protobuf ourselves.
from golang
RUN apt-get update && apt-get install net-tools
RUN wget https://repo1.maven.org/maven2/com/google/protobuf/protoc/3.0.0-beta-2/protoc-3.0.0-beta-2-linux-x86_64.exe -O /usr/local/bin/protoc && chmod +x /usr/local/bin/protoc
RUN go get -u github.com/golang/protobuf/proto github.com/golang/protobuf/protoc-gen-go 
ADD . /go/src/github.com/otm/limes
WORKDIR /go/src/github.com/otm/limes
RUN go clean
RUN go generate
RUN go get
RUN go build

CMD ["./ims", "status"]
