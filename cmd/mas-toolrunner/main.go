//go:build linux

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/voocel/mas/executor/sandbox"
	"github.com/voocel/mas/executor/sandbox/policy"
	"github.com/voocel/mas/tools"
	"github.com/voocel/mas/tools/builtin"
)

const (
	afVsock       = 40
	vmaddrCIDAny  = ^uint32(0)
	maxFrameBytes = 1 << 20
)

type sockaddrVM struct {
	CID  uint32
	Port uint32
}

type rawSockaddrVM struct {
	Family    uint16
	Reserved1 uint16
	Port      uint32
	CID       uint32
	Zero      [4]byte
}

func (sa *sockaddrVM) sockaddr() (unsafe.Pointer, int, error) {
	if sa.Port == 0 {
		return nil, 0, errors.New("vsock port is required")
	}
	rsa := &rawSockaddrVM{
		Family: afVsock,
		Port:   sa.Port,
		CID:    sa.CID,
	}
	return unsafe.Pointer(rsa), int(unsafe.Sizeof(*rsa)), nil
}

func (sa *sockaddrVM) family() int {
	return afVsock
}

type vsockListener struct {
	fd int
}

func listenVsock(port uint32) (*vsockListener, error) {
	fd, err := syscall.Socket(afVsock, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}
	if err := syscall.Bind(fd, &sockaddrVM{CID: vmaddrCIDAny, Port: port}); err != nil {
		_ = syscall.Close(fd)
		return nil, err
	}
	if err := syscall.Listen(fd, 128); err != nil {
		_ = syscall.Close(fd)
		return nil, err
	}
	return &vsockListener{fd: fd}, nil
}

func (l *vsockListener) Accept() (*os.File, error) {
	nfd, _, err := syscall.Accept(l.fd)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(nfd), "vsock"), nil
}

func (l *vsockListener) Close() error {
	if l == nil || l.fd <= 0 {
		return nil
	}
	return syscall.Close(l.fd)
}

func main() {
	port := flag.Uint("port", 5000, "vsock port")
	flag.Parse()

	registry := tools.NewRegistry()
	registerBuiltinTools(registry)

	ln, err := listenVsock(uint32(*port))
	if err != nil {
		fmt.Fprintln(os.Stderr, "vsock listen error:", err)
		os.Exit(1)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintln(os.Stderr, "vsock accept error:", err)
			continue
		}
		go handleConn(conn, registry)
	}
}

func handleConn(conn *os.File, registry *tools.Registry) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), maxFrameBytes)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			msg := "invalid request"
			if errors.Is(err, bufio.ErrTooLong) {
				msg = "request too large"
			}
			writeResp(conn, sandbox.ExecuteToolResponse{
				ToolCallID: "",
				Status:     sandbox.StatusError,
				Error:      &sandbox.ErrorDetail{Code: sandbox.CodeInvalidRequest, Message: msg},
				ExitCode:   1,
			})
		}
		return
	}
	line := scanner.Bytes()

	var req sandbox.ExecuteToolRequest
	if err := json.Unmarshal(line, &req); err != nil {
		writeResp(conn, sandbox.ExecuteToolResponse{
			ToolCallID: req.ToolCallID,
			Status:     sandbox.StatusError,
			Error:      &sandbox.ErrorDetail{Code: sandbox.CodeInvalidRequest, Message: "invalid request"},
			ExitCode:   1,
		})
		return
	}

	tool, ok := registry.Get(req.Tool.Name)
	if !ok {
		writeResp(conn, sandbox.ExecuteToolResponse{
			ToolCallID: req.ToolCallID,
			Status:     sandbox.StatusError,
			Error:      &sandbox.ErrorDetail{Code: sandbox.CodeInvalidRequest, Message: "tool not found"},
			ExitCode:   1,
		})
		return
	}

	if err := policy.ValidateToolPolicy(req.Policy, tool, req.Tool.Args); err != nil {
		writeResp(conn, sandbox.ExecuteToolResponse{
			ToolCallID: req.ToolCallID,
			Status:     sandbox.StatusError,
			Error:      &sandbox.ErrorDetail{Code: sandbox.CodePolicyDenied, Message: err.Error()},
			ExitCode:   1,
		})
		return
	}

	execCtx := context.Background()
	if req.Policy.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(execCtx, req.Policy.Timeout)
		defer cancel()
	}
	start := time.Now()
	result, err := tool.Execute(execCtx, req.Tool.Args)
	resp := sandbox.ExecuteToolResponse{
		ToolCallID: req.ToolCallID,
		Status:     sandbox.StatusOK,
		Result:     result,
		Usage:      &sandbox.Usage{CPUMs: int(time.Since(start).Milliseconds())},
	}
	if err != nil {
		resp.Status = sandbox.StatusError
		resp.Error = &sandbox.ErrorDetail{Code: sandbox.CodeToolFailed, Message: err.Error()}
		resp.ExitCode = 1
	}
	writeResp(conn, resp)
}

func writeResp(conn *os.File, resp sandbox.ExecuteToolResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = conn.Write(append(data, '\n'))
}

func registerBuiltinTools(registry *tools.Registry) {
	_ = registry.Register(builtin.NewCalculator())
	_ = registry.Register(builtin.NewFileSystemTool(nil, 0))
	_ = registry.Register(builtin.NewHTTPClientTool(0))
	_ = registry.Register(builtin.NewWebSearchTool(""))
	_ = registry.Register(builtin.NewFetchTool(0))
}
