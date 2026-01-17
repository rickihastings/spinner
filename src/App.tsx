import React from 'react';
import { Text, Box } from 'ink';
import { Setup } from './commands/Setup.js';

export interface AppProps {
  command?: string;
  flags: {
    name?: string;
    nodeVersion?: string;
    jvmUrl?: string;
    help?: boolean;
    version?: boolean;
  };
}

const HELP_TEXT = `
docker-sandbox - CLI tool for running code in isolated Docker containers

USAGE:
  docker-sandbox setup --name <name> --jvm-url <url> [--node-version <version>]

COMMANDS:
  setup    Build a Docker sandbox image with JDK and Node.js

OPTIONS:
  --name <name>              Name for the Docker image (required)
  --jvm-url <url>            URL to download JDK tarball (required)
  --node-version <version>   Node.js version (default: 20)
  --help                     Show this help message
  --version                  Show version information

EXAMPLES:
  docker-sandbox setup --name my-sandbox --jvm-url https://download.oracle.com/java/25/latest/jdk-25_linux-aarch64_bin.tar.gz
  docker-sandbox setup --name my-sandbox --jvm-url https://download.oracle.com/java/25/latest/jdk-25_linux-x64_bin.tar.gz --node-version 20

NOTES:
  - You must provide a JVM URL compatible with the target container architecture
  - For ARM64/aarch64: use jdk-*_linux-aarch64_bin.tar.gz
  - For x86_64: use jdk-*_linux-x64_bin.tar.gz
`;

export const App: React.FC<AppProps> = ({ command, flags }) => {
  if (flags.help) {
    return <Text>{HELP_TEXT}</Text>;
  }

  if (flags.version) {
    return <Text>docker-sandbox version 0.1.0</Text>;
  }

  if (command === 'setup') {
    if (!flags.name || !flags.jvmUrl) {
      return (
        <Box flexDirection="column">
          <Text color="red">Error: Missing required flag{!flags.name && !flags.jvmUrl ? 's' : ''}: {!flags.name ? '--name' : ''}{!flags.name && !flags.jvmUrl ? ' and ' : ''}{!flags.jvmUrl ? '--jvm-url' : ''}</Text>
          <Text>Usage: docker-sandbox setup --name &lt;name&gt; --jvm-url &lt;url&gt; [--node-version &lt;version&gt;]</Text>
        </Box>
      );
    }

    // Default to Node.js LTS version 20 if not specified
    const nodeVersion = flags.nodeVersion || '20';

    return <Setup name={flags.name} nodeVersion={nodeVersion} jvmUrl={flags.jvmUrl} />;
  }

  return (
    <Box flexDirection="column">
      <Text color="red">Unknown command: {command || '(none)'}</Text>
      <Text>Run with --help for usage information</Text>
    </Box>
  );
};
