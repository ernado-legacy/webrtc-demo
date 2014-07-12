FROM cydev/go
RUN go get -u -v github.com/ernado/webrtc-demo
EXPOSE 5555
WORKDIR /opt/go/src/github.com/ernado/webrtc-demo
ENTRYPOINT ["webrtc-demo"]
