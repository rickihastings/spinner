FROM golang:1.25

WORKDIR /workspace

# Install git for go module operations
RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

CMD ["/bin/bash"]
