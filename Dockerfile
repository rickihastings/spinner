FROM ubuntu:22.04

WORKDIR /workspace

# Install essential tools
RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

CMD ["/bin/bash"]
