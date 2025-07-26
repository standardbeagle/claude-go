package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	mu          sync.RWMutex
	sessions    map[string]*Session
	defaultOpts *Options
}

type Options struct {
	Model            string            `json:"model,omitempty"`
	MaxTurns         int               `json:"max_turns,omitempty"`
	Debug            bool              `json:"debug,omitempty"`
	Verbose          bool              `json:"verbose,omitempty"`
	PermissionMode   string            `json:"permission_mode,omitempty"`
	AllowedTools     []string          `json:"allowed_tools,omitempty"`
	DisallowedTools  []string          `json:"disallowed_tools,omitempty"`
	SystemPrompt     string            `json:"system_prompt,omitempty"`
	MCPConfig        string            `json:"mcp_config,omitempty"`
	AddDirectories   []string          `json:"add_directories,omitempty"`
	Environment      map[string]string `json:"environment,omitempty"`
	WorkingDirectory string            `json:"working_directory,omitempty"`
	Interactive      bool              `json:"interactive,omitempty"` // Use interactive mode instead of -p
}

type Session struct {
	ID       string
	client   *Client
	ctx      context.Context
	cancel   context.CancelFunc
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	options  *Options
	messages chan *Message
	errors   chan error
	closed   bool
	mu       sync.RWMutex
}

type Message struct {
	Type      string          `json:"type"`
	Content   string          `json:"content,omitempty"`
	Role      string          `json:"role,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	Error     string          `json:"error,omitempty"`
}

type QueryRequest struct {
	Prompt    string   `json:"prompt"`
	Options   *Options `json:"options,omitempty"`
	SessionID string   `json:"session_id,omitempty"`
}

type QueryResponse struct {
	SessionID string        `json:"session_id"`
	Messages  chan *Message `json:"-"`
	Errors    chan error    `json:"-"`
}

func New(opts *Options) *Client {
	if opts == nil {
		opts = &Options{
			PermissionMode: "bypassPermissions",
			Debug:          false,
			Verbose:        false,
			Interactive:    true, // Default to interactive mode for multi-turn support
		}
	}

	return &Client{
		sessions:    make(map[string]*Session),
		defaultOpts: opts,
	}
}

func (c *Client) Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	opts := c.mergeOptions(req.Options)
	session, err := c.createSession(ctx, sessionID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	go session.handleIO()

	// Give Claude a moment to start up before sending the prompt
	time.Sleep(100 * time.Millisecond)

	if err := session.sendPrompt(req.Prompt); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to send prompt: %w", err)
	}

	return &QueryResponse{
		SessionID: sessionID,
		Messages:  session.messages,
		Errors:    session.errors,
	}, nil
}

func (c *Client) GetSession(sessionID string) (*Session, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	session, exists := c.sessions[sessionID]
	return session, exists
}

func (c *Client) CloseSession(sessionID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if session, exists := c.sessions[sessionID]; exists {
		session.Close()
		delete(c.sessions, sessionID)
		return nil
	}

	return fmt.Errorf("session %s not found", sessionID)
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for sessionID, session := range c.sessions {
		session.Close()
		delete(c.sessions, sessionID)
	}

	return nil
}

func (c *Client) createSession(ctx context.Context, sessionID string, opts *Options) (*Session, error) {
	sessionCtx, cancel := context.WithCancel(ctx)

	args := c.buildArgs(opts)

	// Add session ID for interactive mode
	if opts.Interactive {
		args = append(args, "--session-id", sessionID)
	}

	cmd := exec.CommandContext(sessionCtx, "claude", args...)

	if opts.WorkingDirectory != "" {
		cmd.Dir = opts.WorkingDirectory
	}

	if opts.Environment != nil {
		env := cmd.Environ()
		for k, v := range opts.Environment {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("failed to start claude command: %w", err)
	}

	session := &Session{
		ID:       sessionID,
		client:   c,
		ctx:      sessionCtx,
		cancel:   cancel,
		cmd:      cmd,
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		options:  opts,
		messages: make(chan *Message, 100),
		errors:   make(chan error, 10),
		closed:   false,
	}

	c.mu.Lock()
	c.sessions[sessionID] = session
	c.mu.Unlock()

	return session, nil
}

func (c *Client) mergeOptions(opts *Options) *Options {
	merged := &Options{}

	if c.defaultOpts != nil {
		*merged = *c.defaultOpts
	}

	if opts != nil {
		if opts.Model != "" {
			merged.Model = opts.Model
		}
		if opts.MaxTurns > 0 {
			merged.MaxTurns = opts.MaxTurns
		}
		if opts.Debug {
			merged.Debug = opts.Debug
		}
		if opts.Verbose {
			merged.Verbose = opts.Verbose
		}
		if opts.PermissionMode != "" {
			merged.PermissionMode = opts.PermissionMode
		}
		if len(opts.AllowedTools) > 0 {
			merged.AllowedTools = opts.AllowedTools
		}
		if len(opts.DisallowedTools) > 0 {
			merged.DisallowedTools = opts.DisallowedTools
		}
		if opts.SystemPrompt != "" {
			merged.SystemPrompt = opts.SystemPrompt
		}
		if opts.MCPConfig != "" {
			merged.MCPConfig = opts.MCPConfig
		}
		if len(opts.AddDirectories) > 0 {
			merged.AddDirectories = opts.AddDirectories
		}
		if opts.Environment != nil {
			if merged.Environment == nil {
				merged.Environment = make(map[string]string)
			}
			for k, v := range opts.Environment {
				merged.Environment[k] = v
			}
		}
		if opts.WorkingDirectory != "" {
			merged.WorkingDirectory = opts.WorkingDirectory
		}
	}

	return merged
}

func (c *Client) buildArgs(opts *Options) []string {
	var args []string

	if opts.Interactive {
		args = []string{"--dangerously-skip-permissions"}
		// Don't use -p for interactive mode
	} else {
		args = []string{
			"-p",
			"--dangerously-skip-permissions",
		}
	}

	if opts.Debug {
		args = append(args, "--debug")
	}

	if opts.Verbose {
		args = append(args, "--verbose")
	}

	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}

	if opts.PermissionMode != "" {
		args = append(args, "--permission-mode", opts.PermissionMode)
	}

	if len(opts.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(opts.AllowedTools, ","))
	}

	if len(opts.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools", strings.Join(opts.DisallowedTools, ","))
	}

	if opts.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", opts.SystemPrompt)
	}

	if opts.MCPConfig != "" {
		args = append(args, "--mcp-config", opts.MCPConfig)
	}

	if len(opts.AddDirectories) > 0 {
		for _, dir := range opts.AddDirectories {
			args = append(args, "--add-dir", dir)
		}
	}

	return args
}

func (s *Session) sendPrompt(prompt string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	// For simple -p mode, just write the prompt directly
	_, err := s.stdin.Write([]byte(prompt + "\n"))
	if err != nil {
		return fmt.Errorf("failed to write prompt: %w", err)
	}

	return nil
}

func (s *Session) SendMessage(content string) error {
	return s.sendPrompt(content)
}

func (s *Session) handleIO() {
	defer func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()

		close(s.messages)
		close(s.errors)
	}()

	go s.handleStderr()
	s.handleStdout()
}

func (s *Session) handleStdout() {
	scanner := bufio.NewScanner(s.stdout)
	var contentBuffer strings.Builder
	messagesSent := false

	for scanner.Scan() {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Temporarily show all output for debugging
		// if strings.HasPrefix(line, "[DEBUG]") ||
		//    strings.HasPrefix(line, "[ERROR]") ||
		//    strings.HasPrefix(line, "[MCP]") ||
		//    strings.HasPrefix(line, "Error:") ||
		//    strings.TrimSpace(line) == "" {
		//	continue
		// }

		contentBuffer.WriteString(line)
		contentBuffer.WriteString("\n")

		// Send incremental content for each meaningful line
		msg := Message{
			Type:      "content",
			Content:   line,
			Timestamp: time.Now(),
		}

		select {
		case s.messages <- &msg:
			messagesSent = true
		case <-s.ctx.Done():
			return
		}
	}

	// Send final accumulated content if we got any messages
	if contentBuffer.Len() > 0 && messagesSent {
		finalMsg := Message{
			Type:      "final",
			Content:   strings.TrimSpace(contentBuffer.String()),
			Timestamp: time.Now(),
		}

		select {
		case s.messages <- &finalMsg:
		case <-s.ctx.Done():
			return
		}
	}

	if err := scanner.Err(); err != nil {
		s.errors <- fmt.Errorf("stdout scanner error: %w", err)
	}
}

func (s *Session) handleStderr() {
	scanner := bufio.NewScanner(s.stderr)

	for scanner.Scan() {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if line != "" {
			s.errors <- fmt.Errorf("stderr: %s", line)
		}
	}

	if err := scanner.Err(); err != nil {
		s.errors <- fmt.Errorf("stderr scanner error: %w", err)
	}
}

func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	s.cancel()

	if s.stdin != nil {
		s.stdin.Close()
	}

	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}

	return nil
}

func (s *Session) Wait() error {
	if s.cmd == nil {
		return fmt.Errorf("no command to wait for")
	}
	return s.cmd.Wait()
}
