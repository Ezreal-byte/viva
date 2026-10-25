package devreload

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/vivaui/viva/internal/reloadwire"
)

type workerProcess struct {
	client  *reloadwire.Client
	config  reloadwire.WindowConfig
	command *exec.Cmd
	wait    chan error
	binary  string
	close   sync.Once
}

func startWorker(ctx context.Context, binary string, arguments []string, output, errorOutput io.Writer, timeout time.Duration) (*workerProcess, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	defer listener.Close()
	tokenData := make([]byte, 32)
	if _, err := rand.Read(tokenData); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(tokenData)
	command := exec.CommandContext(ctx, binary, arguments...)
	command.Env = workerEnvironment(listener.Addr().String(), token)
	command.Stdout = output
	command.Stderr = errorOutput
	if err := command.Start(); err != nil {
		return nil, err
	}
	wait := make(chan error, 1)
	go func() { wait <- command.Wait() }()

	if tcp, ok := listener.(*net.TCPListener); ok {
		_ = tcp.SetDeadline(time.Now().Add(timeout))
	}
	connection, err := listener.Accept()
	if err != nil {
		_ = command.Process.Kill()
		<-wait
		return nil, fmt.Errorf("worker did not connect within %s: %w", timeout, err)
	}
	_ = connection.SetDeadline(time.Now().Add(timeout))
	hello, client, err := reloadwire.Accept(connection, token)
	_ = connection.SetDeadline(time.Time{})
	if err != nil {
		connection.Close()
		_ = command.Process.Kill()
		<-wait
		return nil, err
	}
	return &workerProcess{client: client, config: hello.Config, command: command, wait: wait, binary: binary}, nil
}

func (worker *workerProcess) stop() {
	worker.close.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = worker.client.Shutdown(ctx)
		cancel()
		_ = worker.client.Close()
		select {
		case <-worker.wait:
		case <-time.After(time.Second):
			_ = worker.command.Process.Kill()
			<-worker.wait
		}
		_ = os.Remove(worker.binary)
	})
}

func workerEnvironment(address, token string) []string {
	result := make([]string, 0, len(os.Environ())+2)
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, reloadwire.AddressEnvironment+"=") || strings.HasPrefix(value, reloadwire.TokenEnvironment+"=") {
			continue
		}
		result = append(result, value)
	}
	return append(result, reloadwire.AddressEnvironment+"="+address, reloadwire.TokenEnvironment+"="+token)
}
