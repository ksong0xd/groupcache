# syntax=docker/dockerfile:1

#FROM golang:1.21.0

#faster by using local docker image
FROM cache-client:latest

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY ./ ./
#RUN ls -la /app;sleep 10000

# Build
RUN CGO_ENABLED=0 GOOS=linux go build k8s/example/cmd/main.go

# Optional:
# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/engine/reference/builder/#expose
EXPOSE 8080

# Run
CMD ["./main"]