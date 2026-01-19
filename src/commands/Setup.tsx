import React, { useState, useEffect } from 'react';
import { Text, Box } from 'ink';
import { existsSync } from 'fs';
import { checkPrerequisites, PrerequisiteError } from '../utils/prerequisites.js';
import { buildImage } from '../utils/docker.js';

export interface SetupProps {
  name: string;
  baseImage?: string;
  dockerfile?: string;
}

export const Setup: React.FC<SetupProps> = ({ name, baseImage, dockerfile }) => {
  const [status, setStatus] = useState<'checking' | 'building' | 'success' | 'error'>('checking');
  const [errorMessage, setErrorMessage] = useState<string>('');

  useEffect(() => {
    async function run() {
      try {
        setStatus('checking');
        checkPrerequisites();

        // Validate Dockerfile path if provided
        if (dockerfile && !existsSync(dockerfile)) {
          throw new Error(`Dockerfile not found at path: ${dockerfile}`);
        }

        setStatus('building');
        buildImage({ name, baseImage, dockerfile });

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
  }, [name, baseImage, dockerfile]);

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
        <Text>Building Docker image: spinner:{name}</Text>
      </Box>
    );
  }

  if (status === 'success') {
    return (
      <Box flexDirection="column">
        <Text color="green">✓ Prerequisites checked</Text>
        <Text color="green">✓ Docker image built successfully: spinner:{name}</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      <Text color="red">✗ Error: {errorMessage}</Text>
    </Box>
  );
};
