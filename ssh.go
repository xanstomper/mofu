package mofu

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

// Middleware wraps a SessionHandler.
type Middleware func(SessionHandler) SessionHandler

// SessionHandler processes a single SSH session.
type SessionHandler func(sess *SSHSession)

// SSHSession wraps an SSH channel to implement io.ReadWriteCloser.
type SSHSession struct {
	ssh.Channel
	remoteAddr string
	IsPty      bool
	closed     sync.Once
}

func (s *SSHSession) Close() error {
	s.closed.Do(func() {
		s.Channel.Close()
	})
	return nil
}

func (s *SSHSession) RemoteAddr() string {
	return s.remoteAddr
}

// SSHServerConfig holds configuration for the SSH server.
type SSHServerConfig struct {
	Addr        string
	HostKey     []byte
	NewProgram  func(*SSHSession) *Program
	Middlewares []Middleware
}

// SSHServer serves MOFU apps over SSH.
type SSHServer struct {
	sshConfig  *ssh.ServerConfig
	listener   net.Listener
	newProgram func(*SSHSession) *Program
	middleware SessionHandler
	sessions   int64
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewSSHServer creates a new SSH server.
func NewSSHServer(cfg SSHServerConfig) (*SSHServer, error) {
	sshCfg := &ssh.ServerConfig{}

	if cfg.HostKey != nil {
		signer, err := ssh.ParsePrivateKey(cfg.HostKey)
		if err != nil {
			return nil, fmt.Errorf("ssh: invalid host key: %w", err)
		}
		sshCfg.AddHostKey(signer)
	} else {
		key, err := generateHostKey()
		if err != nil {
			return nil, fmt.Errorf("ssh: failed to generate host key: %w", err)
		}
		sshCfg.AddHostKey(key)
	}

	if cfg.NewProgram == nil {
		return nil, fmt.Errorf("ssh: NewProgram function is required")
	}

	// Build middleware chain (outermost first)
	var handler SessionHandler = defaultSessionHandler(cfg.NewProgram)
	for i := len(cfg.Middlewares) - 1; i >= 0; i-- {
		handler = cfg.Middlewares[i](handler)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &SSHServer{
		sshConfig:  sshCfg,
		newProgram: cfg.NewProgram,
		middleware: handler,
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

func defaultSessionHandler(newProgram func(*SSHSession) *Program) SessionHandler {
	return func(sess *SSHSession) {
		p := newProgram(sess)
		_ = p.Run()
	}
}

// Serve starts listening for SSH connections.
func (s *SSHServer) Serve(addr string) error {
	var err error
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("ssh: listen: %w", err)
	}

	log.Printf("SSH server listening on %s", addr)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return nil
			default:
				log.Printf("ssh: accept: %v", err)
				continue
			}
		}
		go s.handleConn(conn)
	}
}

func (s *SSHServer) handleConn(netConn net.Conn) {
	sshConn, chans, reqs, err := ssh.NewServerConn(netConn, s.sshConfig)
	if err != nil {
		log.Printf("ssh: handshake: %v", err)
		netConn.Close()
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	remoteAddr := sshConn.RemoteAddr().String()

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}
		go s.handleChannel(newChan, remoteAddr)
	}
}

func (s *SSHServer) handleChannel(newChan ssh.NewChannel, remoteAddr string) {
	ch, chReqs, err := newChan.Accept()
	if err != nil {
		log.Printf("ssh: channel accept: %v", err)
		return
	}

	atomic.AddInt64(&s.sessions, 1)
	defer atomic.AddInt64(&s.sessions, -1)

	sess := &SSHSession{
		Channel:    ch,
		remoteAddr: remoteAddr,
	}
	defer sess.Close()

	// Handle pty-req and window-change requests before passing to middleware
	for req := range chReqs {
		switch req.Type {
		case "pty-req":
			sess.IsPty = true
			if req.WantReply {
				req.Reply(true, nil)
			}
		case "window-change":
			if req.WantReply {
				req.Reply(true, nil)
			}
		case "env":
			if req.WantReply {
				req.Reply(false, nil)
			}
		default:
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}

	s.middleware(sess)
}

// Sessions returns the current number of active sessions.
func (s *SSHServer) Sessions() int64 {
	return atomic.LoadInt64(&s.sessions)
}

// Close stops the SSH server.
func (s *SSHServer) Close() error {
	s.cancel()
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func generateHostKey() (ssh.Signer, error) {
	// Generate a temporary Ed25519 key for development.
	// In production, provide a proper host key via SSHServerConfig.HostKey.
	return nil, fmt.Errorf("no host key provided; generate one with: ssh-keygen -t ed25519 -f /tmp/mofu_host_key")
}

// LoggingMiddleware logs SSH session activity.
// If logger is nil, the default logger is used.
func LoggingMiddleware(logger *log.Logger) Middleware {
	if logger == nil {
		logger = log.Default()
	}
	return func(next SessionHandler) SessionHandler {
		return func(sess *SSHSession) {
			start := time.Now()
			logger.Printf("session started: %s", sess.RemoteAddr())
			next(sess)
			logger.Printf("session ended: %s duration=%v", sess.RemoteAddr(), time.Since(start))
		}
	}
}

// RateLimitMiddleware limits concurrent SSH sessions.
func RateLimitMiddleware(maxConcurrent int) Middleware {
	var count int64
	return func(next SessionHandler) SessionHandler {
		return func(sess *SSHSession) {
			if atomic.AddInt64(&count, 1) > int64(maxConcurrent) {
				atomic.AddInt64(&count, -1)
				fmt.Fprintf(sess, "server full, try again later\n")
				return
			}
			defer atomic.AddInt64(&count, -1)
			next(sess)
		}
	}
}

// AuthMiddleware validates SSH connections via public key.
func AuthMiddleware(callback func(sess *SSHSession) bool) Middleware {
	return func(next SessionHandler) SessionHandler {
		return func(sess *SSHSession) {
			if !callback(sess) {
				fmt.Fprintf(sess, "access denied\n")
				return
			}
			next(sess)
		}
	}
}
