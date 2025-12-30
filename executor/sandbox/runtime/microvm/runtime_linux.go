//go:build linux

package microvm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/voocel/mas/executor/sandbox"
	"github.com/voocel/mas/executor/sandbox/runtime"
)

type Runtime struct {
	Config Config
	pool   *vmPool
	once   sync.Once
}

var _ runtime.Runtime = (*Runtime)(nil)

func NewRuntime(cfg Config) *Runtime {
	return &Runtime{Config: cfg}
}

func (r *Runtime) CreateSandbox(ctx context.Context, req sandbox.CreateSandboxRequest) (*sandbox.CreateSandboxResponse, error) {
	vm, err := r.reserve(ctx, req.SandboxID)
	if err != nil {
		return nil, err
	}
	return &sandbox.CreateSandboxResponse{SandboxID: vm.sandboxID, Status: sandbox.StatusOK}, nil
}

func (r *Runtime) ExecuteTool(ctx context.Context, req sandbox.ExecuteToolRequest) (*sandbox.ExecuteToolResponse, error) {
	var vm *vmInstance
	if strings.TrimSpace(req.SandboxID) != "" {
		var err error
		vm, err = r.acquireReserved(req.SandboxID)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		vm, err = r.acquire(ctx)
		if err != nil {
			return nil, err
		}
		defer r.release(vm)
	}

	if err := validatePolicy(r.Config, req); err != nil {
		return policyDenied(req.ToolCallID, err.Error()), nil
	}

	execCtx := ctx
	if req.Policy.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, req.Policy.Timeout)
		defer cancel()
	}

	resp, err := runToolRunner(execCtx, r.Config, vm, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (r *Runtime) DestroySandbox(ctx context.Context, req sandbox.DestroySandboxRequest) (*sandbox.DestroySandboxResponse, error) {
	_ = ctx
	if r.pool != nil {
		r.pool.stopBySandboxID(req.SandboxID)
	}
	return &sandbox.DestroySandboxResponse{Status: sandbox.StatusOK}, nil
}

func (r *Runtime) acquire(ctx context.Context) (*vmInstance, error) {
	if r == nil {
		return nil, errors.New("runtime is nil")
	}
	if err := r.ensurePool(ctx); err != nil {
		return nil, err
	}
	return r.pool.get(ctx)
}

func (r *Runtime) release(vm *vmInstance) {
	if r.pool != nil && vm != nil {
		r.pool.put(vm)
	}
}

func (r *Runtime) reserve(ctx context.Context, sandboxID string) (*vmInstance, error) {
	if r == nil {
		return nil, errors.New("runtime is nil")
	}
	if err := r.ensurePool(ctx); err != nil {
		return nil, err
	}
	return r.pool.reserve(ctx, sandboxID)
}

func (r *Runtime) acquireReserved(sandboxID string) (*vmInstance, error) {
	if r == nil || r.pool == nil {
		return nil, errors.New("runtime is nil")
	}
	return r.pool.getReserved(sandboxID)
}

func (r *Runtime) ensurePool(ctx context.Context) error {
	var initErr error
	r.once.Do(func() {
		cfg, err := normalizeConfig(r.Config)
		if err != nil {
			initErr = err
			return
		}
		r.Config = cfg
		size := r.Config.Pool.Size
		if size <= 0 {
			size = 1
		}
		pool := newVMPool(size, r.Config)
		if err := pool.warm(ctx); err != nil {
			initErr = err
			return
		}
		r.pool = pool
	})
	return initErr
}

type vmPool struct {
	mu       sync.Mutex
	cfg      Config
	size     int
	ready    chan *vmInstance
	reserved map[string]*vmInstance
	all      map[string]*vmInstance
	retire   map[string]bool
	closed   bool
}

func newVMPool(size int, cfg Config) *vmPool {
	return &vmPool{
		cfg:      cfg,
		size:     size,
		ready:    make(chan *vmInstance, size),
		reserved: make(map[string]*vmInstance),
		all:      make(map[string]*vmInstance),
		retire:   make(map[string]bool),
	}
}

func (p *vmPool) warm(ctx context.Context) error {
	started := make([]*vmInstance, 0, p.size)
	for i := 0; i < p.size; i++ {
		vm, err := startVM(ctx, p.cfg)
		if err != nil {
			for _, inst := range started {
				p.removeVM(inst)
				_ = stopVM(inst)
			}
			return err
		}
		p.addVM(vm)
		started = append(started, vm)
		p.ready <- vm
	}
	return nil
}

func (p *vmPool) get(ctx context.Context) (*vmInstance, error) {
	if p == nil {
		return nil, errors.New("vm pool is nil")
	}
	for {
		select {
		case vm := <-p.ready:
			if vm == nil {
				continue
			}
			if p.shouldRetire(vm) {
				p.removeVM(vm)
				_ = stopVM(vm)
				continue
			}
			return vm, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (p *vmPool) put(vm *vmInstance) {
	if p == nil || vm == nil {
		return
	}
	if p.shouldRetire(vm) {
		p.removeVM(vm)
		_ = stopVM(vm)
		return
	}
	select {
	case p.ready <- vm:
	default:
		p.removeVM(vm)
		_ = stopVM(vm)
	}
}

func (p *vmPool) reserve(ctx context.Context, sandboxID string) (*vmInstance, error) {
	if p == nil {
		return nil, errors.New("vm pool is nil")
	}
	if strings.TrimSpace(sandboxID) != "" {
		p.mu.Lock()
		if _, ok := p.reserved[sandboxID]; ok {
			p.mu.Unlock()
			return nil, errors.New("sandbox already exists")
		}
		p.mu.Unlock()
	}
	vm, err := p.get(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(sandboxID) == "" {
		sandboxID = vm.id
	}
	vm.sandboxID = sandboxID
	p.mu.Lock()
	p.reserved[sandboxID] = vm
	p.mu.Unlock()
	return vm, nil
}

func (p *vmPool) getReserved(sandboxID string) (*vmInstance, error) {
	if p == nil {
		return nil, errors.New("vm pool is nil")
	}
	if strings.TrimSpace(sandboxID) == "" {
		return nil, errors.New("sandbox id is required")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	vm, ok := p.reserved[sandboxID]
	if !ok {
		return nil, errors.New("sandbox not found")
	}
	return vm, nil
}

func (p *vmPool) stopBySandboxID(sandboxID string) {
	if p == nil || strings.TrimSpace(sandboxID) == "" {
		return
	}
	p.mu.Lock()
	if vm, ok := p.reserved[sandboxID]; ok {
		delete(p.reserved, sandboxID)
		delete(p.all, vm.id)
		p.mu.Unlock()
		_ = stopVM(vm)
		return
	}
	p.retire[sandboxID] = true
	p.mu.Unlock()

	p.drainStop(sandboxID)
}

func (p *vmPool) drainStop(id string) {
	kept := make([]*vmInstance, 0, len(p.ready))
	for {
		select {
		case vm := <-p.ready:
			if vm == nil {
				continue
			}
			if vm.id == id || vm.sandboxID == id {
				p.removeVM(vm)
				_ = stopVM(vm)
				continue
			}
			kept = append(kept, vm)
		default:
			for _, vm := range kept {
				p.ready <- vm
			}
			return
		}
	}
}

func (p *vmPool) addVM(vm *vmInstance) {
	if p == nil || vm == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.all[vm.id] = vm
}

func (p *vmPool) removeVM(vm *vmInstance) {
	if p == nil || vm == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.all, vm.id)
	if vm.sandboxID != "" {
		delete(p.reserved, vm.sandboxID)
	}
	delete(p.retire, vm.id)
}

func (p *vmPool) shouldRetire(vm *vmInstance) bool {
	if p == nil || vm == nil {
		return true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return true
	}
	if vm.sandboxID != "" {
		if _, ok := p.retire[vm.sandboxID]; ok {
			delete(p.retire, vm.sandboxID)
			return true
		}
	}
	if _, ok := p.retire[vm.id]; ok {
		delete(p.retire, vm.id)
		return true
	}
	return false
}

type vmInstance struct {
	id        string
	sandboxID string
	apiSock   string
	vsockPath string
	workdir   string
	process   *os.Process
	client    *fcClient
}

func startVM(ctx context.Context, cfg Config) (*vmInstance, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	workdir, err := os.MkdirTemp("", "mas-microvm-*")
	if err != nil {
		return nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(workdir)
	}

	instanceID := filepath.Base(workdir)

	apiSock := expandPath(cfg.APISocket, instanceID)
	if apiSock == "" {
		apiSock = filepath.Join(workdir, "firecracker.sock")
	}
	logPath := expandPath(cfg.LogPath, instanceID)
	if logPath == "" {
		logPath = filepath.Join(workdir, "firecracker.log")
	}
	metricsPath := expandPath(cfg.MetricsPath, instanceID)
	if metricsPath == "" {
		metricsPath = filepath.Join(workdir, "firecracker.metrics")
	}

	args := []string{"--api-sock", apiSock, "--log-path", logPath, "--metrics-path", metricsPath}
	cmd := exec.CommandContext(ctx, cfg.FirecrackerBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		cleanup()
		return nil, err
	}

	if err := waitForSocket(ctx, apiSock); err != nil {
		_ = cmd.Process.Kill()
		cleanup()
		return nil, err
	}

	instanceCfg := cfg
	instanceCfg.APISocket = apiSock
	instanceCfg.LogPath = logPath
	instanceCfg.MetricsPath = metricsPath
	instanceCfg.VSock.UDSPath = expandPath(cfg.VSock.UDSPath, instanceID)
	instanceCfg.Network.TapDevice = expandPath(cfg.Network.TapDevice, instanceID)

	client := newFCClient(apiSock)
	if err := client.configure(instanceCfg); err != nil {
		_ = cmd.Process.Kill()
		cleanup()
		return nil, err
	}
	if err := client.startInstance(); err != nil {
		_ = cmd.Process.Kill()
		cleanup()
		return nil, err
	}

	vm := &vmInstance{
		id:        instanceID,
		apiSock:   apiSock,
		vsockPath: instanceCfg.VSock.UDSPath,
		workdir:   workdir,
		process:   cmd.Process,
		client:    client,
	}
	return vm, nil
}

func stopVM(vm *vmInstance) error {
	if vm == nil || vm.process == nil {
		return nil
	}
	_ = vm.process.Kill()
	_, _ = vm.process.Wait()
	if vm.workdir != "" {
		_ = os.RemoveAll(vm.workdir)
	}
	return nil
}

func waitForSocket(ctx context.Context, path string) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return errors.New("firecracker api socket timeout")
		case <-ticker.C:
			if _, err := os.Stat(path); err == nil {
				return nil
			}
		}
	}
}

type fcClient struct {
	socketPath string
	client     *http.Client
}

func newFCClient(socketPath string) *fcClient {
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}
	return &fcClient{
		socketPath: socketPath,
		client:     &http.Client{Transport: transport},
	}
}

func (c *fcClient) configure(cfg Config) error {
	if err := c.put("/machine-config", map[string]any{
		"vcpu_count":   cfg.Machine.VCPUCount,
		"mem_size_mib": cfg.Machine.MemMiB,
	}); err != nil {
		return err
	}
	bootArgs := cfg.BootArgs
	if strings.TrimSpace(bootArgs) == "" {
		bootArgs = "console=ttyS0 reboot=k panic=1 pci=off"
	}
	if err := c.put("/boot-source", map[string]any{
		"kernel_image_path": cfg.KernelImage,
		"boot_args":         bootArgs,
	}); err != nil {
		return err
	}
	if err := c.put("/drives/rootfs", map[string]any{
		"drive_id":       "rootfs",
		"path_on_host":   cfg.RootFS,
		"is_root_device": true,
		"is_read_only":   true,
	}); err != nil {
		return err
	}
	if cfg.Network.Enabled {
		if err := c.put("/network-interfaces/eth0", networkPayload(cfg)); err != nil {
			return err
		}
	}
	if cfg.VSock.CID != 0 && cfg.VSock.UDSPath != "" {
		if err := c.put("/vsock", map[string]any{
			"guest_cid": cfg.VSock.CID,
			"uds_path":  cfg.VSock.UDSPath,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c *fcClient) startInstance() error {
	return c.put("/actions", map[string]any{"action_type": "InstanceStart"})
}

func (c *fcClient) put(path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, "http://unix"+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("firecracker api error: %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return nil
}

func runToolRunner(ctx context.Context, cfg Config, vm *vmInstance, req sandbox.ExecuteToolRequest) (*sandbox.ExecuteToolResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.ToolRunner.Command) == "" {
		return nil, errors.New("tool runner command is required")
	}
	cmd := exec.CommandContext(ctx, cfg.ToolRunner.Command, expandArgs(cfg.ToolRunner.Args, vm)...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("MAS_FIRECRACKER_SOCKET=%s", vm.apiSock),
		fmt.Sprintf("MAS_VSOCK_UDS=%s", vm.vsockPath),
		fmt.Sprintf("MAS_VSOCK_CID=%d", cfg.VSock.CID),
		fmt.Sprintf("MAS_VSOCK_PORT=%d", cfg.VSock.Port),
	)
	cmd.Stdin = bytes.NewReader(append(data, '\n'))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tool runner error: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return parseToolResponse(out)
}

func parseToolResponse(output []byte) (*sandbox.ExecuteToolResponse, error) {
	lines := strings.Split(string(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var resp sandbox.ExecuteToolResponse
		if err := json.Unmarshal([]byte(line), &resp); err == nil {
			return &resp, nil
		}
	}
	return nil, errors.New("invalid tool runner response")
}

func expandArgs(args []string, vm *vmInstance) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		arg = strings.ReplaceAll(arg, "{api_socket}", vm.apiSock)
		arg = strings.ReplaceAll(arg, "{vsock_uds}", vm.vsockPath)
		out = append(out, arg)
	}
	return out
}

func expandPath(value, instanceID string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return strings.ReplaceAll(value, "{id}", instanceID)
}

func normalizeConfig(cfg Config) (Config, error) {
	if cfg.Machine.VCPUCount == 0 {
		cfg.Machine.VCPUCount = 1
	}
	if cfg.Machine.MemMiB == 0 {
		cfg.Machine.MemMiB = 512
	}
	return cfg, validateConfig(cfg)
}

func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.FirecrackerBin) == "" {
		return errors.New("firecracker_bin is required")
	}
	if strings.TrimSpace(cfg.KernelImage) == "" || strings.TrimSpace(cfg.RootFS) == "" {
		return errors.New("kernel_image and rootfs are required")
	}
	if strings.TrimSpace(cfg.VSock.UDSPath) == "" || cfg.VSock.Port == 0 || cfg.VSock.CID == 0 {
		return errors.New("vsock cid, port, and uds_path are required")
	}
	if cfg.Network.Enabled && strings.TrimSpace(cfg.Network.TapDevice) == "" {
		return errors.New("tap_device is required when network is enabled")
	}
	if cfg.Pool.Size > 1 {
		if cfg.APISocket != "" && !strings.Contains(cfg.APISocket, "{id}") {
			return errors.New("api_socket must include {id} when pool.size > 1")
		}
		if !strings.Contains(cfg.VSock.UDSPath, "{id}") {
			return errors.New("vsock.uds_path must include {id} when pool.size > 1")
		}
		if cfg.Network.Enabled && !strings.Contains(cfg.Network.TapDevice, "{id}") {
			return errors.New("network.tap_device must include {id} when pool.size > 1")
		}
	}
	return nil
}

func validatePolicy(cfg Config, req sandbox.ExecuteToolRequest) error {
	if req.Policy.Network.Enabled && !cfg.Network.Enabled {
		return errors.New("network is disabled by runtime")
	}
	return nil
}

func policyDenied(toolCallID, msg string) *sandbox.ExecuteToolResponse {
	return &sandbox.ExecuteToolResponse{
		ToolCallID: toolCallID,
		Status:     sandbox.StatusError,
		Error:      &sandbox.ErrorDetail{Code: sandbox.CodePolicyDenied, Message: msg},
		ExitCode:   1,
	}
}

func networkPayload(cfg Config) map[string]any {
	payload := map[string]any{
		"iface_id":      "eth0",
		"host_dev_name": cfg.Network.TapDevice,
	}
	if strings.TrimSpace(cfg.Network.MacAddress) != "" {
		payload["guest_mac"] = cfg.Network.MacAddress
	}
	return payload
}
