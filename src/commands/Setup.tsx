import React, { useState, useEffect } from 'react';
import { Text, Box } from 'ink';
import { checkPrerequisites, PrerequisiteError } from '../utils/prerequisites.js';
import { buildImage } from '../utils/docker.js';

export interface SetupProps {
  name: string;
  nodeVersion: string;
  jvmUrl: string;
}

export const Setup: React.FC<SetupProps> = ({ name, nodeVersion, jvmUrl }) => {
  const [status, setStatus] = useState<'checking' | 'building' | 'success' | 'error'>('checking');
  const [errorMessage, setErrorMessage] = useState<string>('');

  useEffect(() => {
    async function run() {
      try {
        setStatus('checking');
        checkPrerequisites();

        setStatus('building');
        buildImage({ name, nodeVersion, jvmUrl });

        setStatus('success');
      } catch (error) {
        setStatus('error');
        if (error instanceof PrerequisiteError) {
          setErrorMessage(error.message);
        } else if (error instanceof Error) {
          setErrorMessage(error.message);
        } else {
          setErrorMessage('Unknown error occurred');
        }
      }
    }

    run();
  }, [name, nodeVersion, jvmUrl]);

  if (status === 'checking') {
    return (
      <Box flexDirection="column">
        <Text>Checking prerequisites...</Text>
      </Box>
    );
  }

  if (status === 'building') {
    return (
      <Box flexDirection="column">
        <Text color="green">✓ Prerequisites checked</Text>
        <Text>Building Docker image: docker-sandbox:{name}</Text>
      </Box>
    );
  }

  if (status === 'success') {
    return (
      <Box flexDirection="column">
        <Text color="green">✓ Prerequisites checked</Text>
        <Text color="green">✓ Docker image built successfully: docker-sandbox:{name}</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      <Text color="red">✗ Error: {errorMessage}</Text>
    </Box>
  );
};
