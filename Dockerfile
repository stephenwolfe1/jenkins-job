FROM scratch
COPY binary/ca-certificates.crt /etc/ssl/certs/
COPY binary/main /
ENTRYPOINT ["/main"]
