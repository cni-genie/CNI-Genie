FROM alpine:3.7

RUN apk add --no-cache \
      iptables

COPY dist/genie-policy /genie-policy
ENTRYPOINT [ "/genie-policy" ]
