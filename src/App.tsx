import React from 'react';
import { Text, Box } from 'ink';
import { Setup } from './commands/Setup.js';
import { Spin } from './commands/Spin.js';

export interface AppProps {
  command?: string;
  flags: {
    name?: string;
    nodeVersion?: string;
    jvmUrl?: string;
    image?: string;
    repo?: string;
    help?: boolean;
    version?: boolean;
  };
}

const HELP_TEXT = `
Spinner - CLI tool for running code in isolated Docker containers

USAGE:
  spinner setup --name <name> --jvm-url <url> [--node-version <version>]
  spinner spin --image <image> --repo <repo>

COMMANDS:
  setup    Build a Docker sandbox image with JDK and Node.js
  spin     Spin up a development container from a pre-built image

SETUP OPTIONS:
  --name <name>              Name for the Docker image (required)
  --jvm-url <url>            URL to download JDK tarball (required)
  --node-version <version>   Node.js version (default: 20)

SPIN OPTIONS:
  --image <image>            Docker image name (required)
  --repo <repo>              Git SSH clone URL (required)

GENERAL OPTIONS:
  --help                     Show this help message
  --version                  Show version information

EXAMPLES:
  spinner setup --name my-sandbox --jvm-url https://download.oracle.com/java/25/latest/jdk-25_linux-aarch64_bin.tar.gz
  spinner spin --image spinner:my-env --repo git@github.com:octocat/Hello-World.git

NOTES:
  - Setup: You must provide a JVM URL compatible with the target container architecture
  - Setup: For ARM64/aarch64: use jdk-*_linux-aarch64_bin.tar.gz
  - Setup: For x86_64: use jdk-*_linux-x64_bin.tar.gz
  - Spin: SSH agent must be running on host system
  - Spin: Containers are persistent and must be manually stopped/removed
`;

export const App: React.FC<AppProps> = ({ command, flags }) => {
  if (flags.help) {
    return <Text>{HELP_TEXT}</Text>;
  }

  if (flags.version) {
    return <Text>Spinner version 0.1.0</Text>;
  }

  if (command === 'setup') {
    if (!flags.name || !flags.jvmUrl) {
      return (
        <Box flexDirection="column">
          <Text color="red">
            Error: Missing required flag{!flags.name && !flags.jvmUrl ? 's' : ''}:{' '}
            {!flags.name ? '--name' : ''}
            {!flags.name && !flags.jvmUrl ? ' and ' : ''}
            {!flags.jvmUrl ? '--jvm-url' : ''}
          </Text>
          <Text>
            Usage: spinner setup --name &lt;name&gt; --jvm-url &lt;url&gt; [--node-version
            &lt;version&gt;]
          </Text>
        </Box>
      );
    }

    // Default to Node.js LTS version 20 if not specified
    const nodeVersion = flags.nodeVersion || '20';

    return <Setup name={flags.name} nodeVersion={nodeVersion} jvmUrl={flags.jvmUrl} />;
  }

  if (command === 'spin') {
    if (!flags.image) {
      process.exitCode = 1;
      return (
        <Box flexDirection="column">
          <Text color="red">Error: --image flag is required</Text>
          <Text>Usage: spinner spin --image &lt;image&gt; --repo &lt;repo&gt;</Text>
        </Box>
      );
    }

    if (!flags.repo) {
      process.exitCode = 1;
      return (
        <Box flexDirection="column">
          <Text color="red">Error: --repo flag is required</Text>
          <Text>Usage: spinner spin --image &lt;image&gt; --repo &lt;repo&gt;</Text>
        </Box>
      );
    }

    return <Spin image={flags.image} repo={flags.repo} />;
  }

  return (
    <Box flexDirection="column">
      <Text color="red">Unknown command: {command || '(none)'}</Text>
      <Text>Run with --help for usage information</Text>
    </Box>
  );
};
