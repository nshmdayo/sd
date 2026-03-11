FROM golang:1.26

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
 && rm -rf /var/lib/apt/lists/*

# Install golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
    sh -s -- -b "$(go env GOPATH)/bin"

WORKDIR /app

# Pre-cache Go modules
COPY go.mod go.sum ./
RUN go mod download

CMD ["bash"]
