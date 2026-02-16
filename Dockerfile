FROM golang:1.24-bookworm

# Install development tools
RUN apt-get update && apt-get install -y \
    git \
    make \
    docker.io \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js and npm packages
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs \
    && npm install -g @fission-ai/openspec

# Install golangci-lint
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.9.0

WORKDIR /home/spinner/workspace

CMD ["/bin/bash"]
