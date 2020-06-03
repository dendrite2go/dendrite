FROM dendrite2go/nix-go-protobuf

RUN go get -u github.com/golang/protobuf/protoc-gen-go
RUN go get google.golang.org/grpc

RUN mkdir /root/bin
RUN curl -L -sS -D - https://github.com/grpc/grpc-web/releases/download/1.0.7/protoc-gen-grpc-web-1.0.7-linux-x86_64 -o ~/bin/protoc-gen-grpc-web
RUN chmod a+x ~/bin/protoc-gen-grpc-web

ENV PATH="${PATH}:/root/bin:/root/go/bin"

CMD /bin/bash