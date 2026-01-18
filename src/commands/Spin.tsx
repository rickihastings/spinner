import React, { useState, useEffect } from 'react';
import { Text, Box } from 'ink';
import { execSync } from 'child_process';
import { existsSync } from 'fs';
import { homedir } from 'os';
import { join } from 'path';

export interface SpinProps {
  image: string;
  repo: string;
}

export const Spin: React.FC<SpinProps> = ({ image, repo }) => {
  const [status, setStatus] = useState<'validating' | 'creating' | 'success' | 'error'>('validating');
  const [errorMessage, setErrorMessage] = useState<string>('');
  const [containerName, setContainerName] = useState<string>('');
  const [warnings, setWarnings] = useState<string[]>([]);

  useEffect(() => {
    async function run() {
      try {
        setStatus('validating');
        const warningsList: string[] = [];

        // Check if Docker image exists
        try {
          execSync(`docker image inspect ${image}`, { stdio: 'ignore' });
        } catch {
          throw new Error(`Docker image '${image}' not found`);
        }

        // Check SSH_AUTH_SOCK
        const sshAuthSock = process.env.SSH_AUTH_SOCK;
        if (!sshAuthSock || !existsSync(sshAuthSock)) {
          throw new Error('SSH agent not running. Start ssh-agent and add your key.');
        }

        // Check ~/.npmrc
        const npmrcPath = join(homedir(), '.npmrc');
        const hasNpmrc = existsSync(npmrcPath);
        if (!hasNpmrc) {
          warningsList.push('~/.npmrc not found, npm will use default registry');
        }

        // Generate container name from repo
        const repoName = repo.split('/').pop()?.replace(/\.git$/, '') || 'sandbox';
        const timestamp = Date.now();
        const generatedName = `${repoName}-${timestamp}`;
        setContainerName(generatedName);

        setStatus('creating');
        setWarnings(warningsList);

        // Build docker run command
        const dockerArgs = [
          'run',
          '-d',
          '--name', generatedName,
          '-v', `${sshAuthSock}:/ssh-agent`,
          '-e', 'SSH_AUTH_SOCK=/ssh-agent',
          '-e', `REPO_URL=${repo}`,
        ];

        // Add .npmrc mount if it exists
        if (hasNpmrc) {
          dockerArgs.push('-v', `${npmrcPath}:/root/.npmrc`);
        }

        // Add image
        dockerArgs.push(image);

        // Execute docker run (no explicit command - uses image's built-in startup.sh)
        try {
          execSync(`docker ${dockerArgs.join(' ')}`, {
            stdio: 'pipe',
            encoding: 'utf-8',
          });
        } catch (error: any) {
          // Container may have started but clone failed
          // Try to get the git error message from container logs
          try {
            const logs = execSync(`docker logs ${generatedName}`, {
              encoding: 'utf-8',
              stdio: 'pipe',
            });
            throw new Error(`Git clone failed: ${logs.trim()}`);
          } catch {
            throw new Error(error.stderr || error.message || 'Failed to create container');
          }
        }

        // Verify container is running
        try {
          const containerStatus = execSync(`docker inspect -f '{{.State.Status}}' ${generatedName}`, {
            encoding: 'utf-8',
            stdio: 'pipe',
          }).trim();

          if (containerStatus !== 'running') {
            // Get logs to show what went wrong
            const logs = execSync(`docker logs ${generatedName}`, {
              encoding: 'utf-8',
              stdio: 'pipe',
            });
            throw new Error(`Container exited. Logs: ${logs.trim()}`);
          }
        } catch (error: any) {
          if (error.message.includes('Container exited')) {
            throw error;
          }
          throw new Error('Failed to verify container status');
        }

        setStatus('success');
      } catch (error) {
        setStatus('error');
        if (error instanceof Error) {
          setErrorMessage(error.message);
        } else {
          setErrorMessage('Unknown error occurred');
        }
      }
    }

    run();
  }, [image, repo]);

  if (status === 'validating') {
    return (
      <Box flexDirection="column">
        <Text>Validating prerequisites...</Text>
      </Box>
    );
  }

  if (status === 'creating') {
    return (
      <Box flexDirection="column">
        <Text color="green">✓ Prerequisites validated</Text>
        {warnings.map((warning, idx) => (
          <Text key={idx} color="yellow">⚠ Warning: {warning}</Text>
        ))}
        <Text>Creating container: {containerName}</Text>
        <Text>Cloning repository...</Text>
      </Box>
    );
  }

  if (status === 'success') {
    return (
      <Box flexDirection="column">
        <Text color="green">✓ Prerequisites validated</Text>
        {warnings.map((warning, idx) => (
          <Text key={idx} color="yellow">⚠ Warning: {warning}</Text>
        ))}
        <Text color="green">✓ Container created successfully: {containerName}</Text>
        <Text></Text>
        <Text>To access: docker exec -it {containerName} bash</Text>
        <Text>To stop: docker stop {containerName}</Text>
        <Text>To remove: docker rm {containerName}</Text>
      </Box>
    );
  }

  // Error state
  process.exitCode = 1;
  return (
    <Box flexDirection="column">
      <Text color="red">✗ Error: {errorMessage}</Text>
    </Box>
  );
};
