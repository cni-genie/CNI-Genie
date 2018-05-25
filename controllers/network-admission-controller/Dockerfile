FROM busybox

COPY dist/network-admission-controller /network-admission-controller
CMD ["/network-admission-controller","--alsologtostderr","--v=4","2>&1"]
