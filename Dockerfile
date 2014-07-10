FROM cydev/image
RUN go get -v github.com/ernado/webrtc-demo
EXPOSE 5555
WORKDIR /opt/go/bin/webrtc-demo
ENTRYPOINT ["/opt/go/bin/webrtc-demo"]
