package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	vsock := flag.String("vsock", "", "vsock UDS path")
	timeout := flag.Duration("timeout", 10*time.Second, "connection timeout")
	flag.Parse()

	if strings.TrimSpace(*vsock) == "" {
		fmt.Fprintln(os.Stderr, "vsock path is required")
		os.Exit(1)
	}

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read stdin error:", err)
		os.Exit(1)
	}
	if len(bytes.TrimSpace(input)) == 0 {
		fmt.Fprintln(os.Stderr, "empty request")
		os.Exit(1)
	}
	if input[len(input)-1] != '\n' {
		input = append(input, '\n')
	}

	conn, err := net.DialTimeout("unix", *vsock, *timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dial error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	if _, err := conn.Write(input); err != nil {
		fmt.Fprintln(os.Stderr, "write error:", err)
		os.Exit(1)
	}

	reader := bufio.NewReader(conn)
	resp, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr, "read error:", err)
		os.Exit(1)
	}

	_, _ = os.Stdout.Write(resp)
}
