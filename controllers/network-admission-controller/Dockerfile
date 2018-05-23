FROM golang:1.8

WORKDIR /go/src
RUN mkdir -p github.com/network-admission-controller
COPY . ./github.com/network-admission-controller
RUN go install github.com/network-admission-controller
CMD ["network-admission-controller","--alsologtostderr","-v=4","2>&1"]
