
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY ./ ./
RUN go build -o main cli/main.go
RUN ls -al

FROM scratch
ARG DOCKER_METADATA_OUTPUT_VERSION=latest
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/main /usr/bin/cdn-s3
ENV PUBLIC_VERSION=$DOCKER_METADATA_OUTPUT_VERSION
CMD [ "cdn-s3" ]
