# nanoservice/protobuf-go was based on go 1.4.2 at the time of
# writing this Dockerfile, which is why we're downloading and building
# protobuf ourselves.
from golang
RUN wget https://github.com/google/protobuf/releases/download/v3.0.0-beta-2/protobuf-cpp-3.0.0-beta-2.zip
RUN apt-get -y update && apt-get -y install net-tools unzip autotools-dev autoconf build-essential libtool
RUN unzip protobuf-cpp-3.0.0-beta-2.zip
WORKDIR protobuf-3.0.0-beta-2
RUN ./autogen.sh
RUN ./configure
RUN make
RUN make install
RUN ldconfig
RUN go get -u github.com/golang/protobuf/proto github.com/golang/protobuf/protoc-gen-go 
ADD . /go/src/github.com/otm/limes
WORKDIR /go/src/github.com/otm/limes
RUN go clean
RUN go generate
RUN go get
RUN go build

CMD ["./ims", "status"]
