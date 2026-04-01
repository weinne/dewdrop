package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var ErrPathNotFound = errors.New("remote path not found")

type RemotePathStore struct {
	mu       sync.Mutex
	filePath string
	cloudDir string
}

func NewRemotePathStore(filePath string, cloudDir string) *RemotePathStore {
	return &RemotePathStore{filePath: filePath, cloudDir: cloudDir}
}

func (s *RemotePathStore) Get(remoteName string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.readAllUnsafe()
	if err != nil {
		return "", err
	}

	if path, ok := entries[remoteName]; ok && strings.TrimSpace(path) != "" {
		return path, nil
	}

	if s.cloudDir != "" {
		return filepath.Join(s.cloudDir, remoteName), nil
	}

	return "", ErrPathNotFound
}

func (s *RemotePathStore) Set(remoteName string, localPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(remoteName) == "" {
		return errors.New("remote name is required")
	}

	entries, err := s.readAllUnsafe()
	if err != nil {
		return err
	}
	entries[remoteName] = localPath

	return s.writeAllUnsafe(entries)
}

func (s *RemotePathStore) Delete(remoteName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.readAllUnsafe()
	if err != nil {
		return err
	}
	delete(entries, remoteName)

	return s.writeAllUnsafe(entries)
}

func (s *RemotePathStore) readAllUnsafe() (map[string]string, error) {
	entries := make(map[string]string)

	file, err := os.Open(s.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return entries, nil
		}
		return nil, fmt.Errorf("open %s: %w", s.filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		sep := strings.Index(line, "|")
		if sep <= 0 {
			continue
		}

		remote := strings.TrimSpace(line[:sep])
		local := strings.TrimSpace(line[sep+1:])
		if remote == "" {
			continue
		}

		entries[remote] = local
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", s.filePath, err)
	}

	return entries, nil
}

func (s *RemotePathStore) writeAllUnsafe(entries map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(s.filePath), "remote-paths-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	for remote, local := range entries {
		if _, err := fmt.Fprintf(tmpFile, "%s|%s\n", remote, local); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return fmt.Errorf("write temp file: %w", err)
		}
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpFile.Name(), s.filePath); err != nil {
		os.Remove(tmpFile.Name())
		return fmt.Errorf("replace %s: %w", s.filePath, err)
	}

	return nil
}
