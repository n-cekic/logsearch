package ssh

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"os"

	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	client *ssh.Client
}

type FileEntry struct {
	Name  string
	IsDir bool
	Size  int64
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Connect(host, port, user, password, keyPath string) error {
	var authMethods []ssh.AuthMethod

	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	if keyPath != "" {
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("unable to read private key: %v", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("unable to parse private key: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("no authentication method provided")
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, verify host key
		Timeout:         5 * time.Second,
	}

	addr := net.JoinHostPort(host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return err
	}

	c.client = client
	return nil
}

func (c *Client) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

// ListDir lists files in the remote directory.
// It uses 'ls -F' to distinguish directories.
func (c *Client) ListDir(path string) ([]FileEntry, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// Use ls -l to get size and type details easily
	// Format: permissions links owner group size date time name
	output, err := session.Output(fmt.Sprintf("ls -lA --time-style=+%%s '%s'", path))
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %v", err)
	}

	var entries []FileEntry
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "total") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 7 {
			continue
		}

		// parts[0] is permissions (drwxr-xr-x)
		// parts[4] is size
		// parts[6...] is name

		isDir := strings.HasPrefix(parts[0], "d")
		size := int64(0)
		fmt.Sscanf(parts[4], "%d", &size)

		name := strings.Join(parts[6:], " ")
		if name == "." || name == ".." {
			continue
		}

		entries = append(entries, FileEntry{
			Name:  name,
			IsDir: isDir,
			Size:  size,
		})
	}

	return entries, nil
}

// ReadFile reads the content of a remote file.
// If the file ends with .gz, it decompresses it.
func (c *Client) ReadFile(path string) ([]byte, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	output, err := session.Output(fmt.Sprintf("cat '%s'", path))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	if strings.HasSuffix(path, ".gz") {
		gr, err := gzip.NewReader(bytes.NewReader(output))
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %v", err)
		}
		defer gr.Close()

		data, err := io.ReadAll(gr)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress gzip data: %v", err)
		}
		return data, nil
	}

	return output, nil
}
