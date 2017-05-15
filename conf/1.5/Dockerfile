FROM busybox

ADD dist/genie /opt/cni/bin/genie
ADD conf/1.5/launch.sh /launch.sh

ENV PATH=$PATH:/opt/cni/bin
VOLUME /opt/cni
WORKDIR /opt/cni/bin
CMD ["/opt/cni/bin/genie"]
