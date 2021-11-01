FROM alpine AS builder

# Install SSL ca certificates.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache ca-certificates && update-ca-certificates

FROM scratch

# Import from builder.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ADD /o-neko-catnip /app/
ADD /config/application-default.yaml /app/config/
ADD /public /app/public

WORKDIR /app

CMD ["/app/o-neko-catnip", "server"]
