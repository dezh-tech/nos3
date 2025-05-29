# --------------------------
#! Stage 1: Build the Go binary
# --------------------------
FROM golang:1.23.3-alpine AS builder

WORKDIR /app

#* Install required build tools
RUN apk --no-cache add build-base git

#* Cache dependencies by copying go.mod and go.sum first
COPY go.mod go.sum ./
RUN go mod download

#* Copy the rest of the application source code
COPY . .

#* Build the Go binary using the provided Makefile
RUN make build


# --------------------------
#! Stage 2: Create the runtime image
# --------------------------
FROM alpine:latest

WORKDIR /app

#* Define build-time arguments and environment variables
ARG MINIO_ROOT_USER
ENV MINIO_ROOT_USER=${MINIO_ROOT_USER}

ARG MINIO_ROOT_PASSWORD
ENV MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD}

ARG DATABASE_URI
ENV DATABASE_URI=${DATABASE_URI}

ARG BROKER_URI
ENV BROKER_URI=${BROKER_URI}

ARG DEFAULT_NOS3_ADDRESS
ENV DEFAULT_NOS3_ADDRESS=${DEFAULT_NOS3_ADDRESS}

#* Copy the compiled binary from the builder stage
COPY --from=builder /app/build/nos3 .
COPY --from=builder /app/config/config.yml .

#* Set the entrypoint to run the application
ENTRYPOINT ["./nos3", "run", "./config.yml"]
