package mofu

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

type Middleware func(SessionHandler) SessionHandler

type SessionHandler func(sess *SSHSession)

type SSHSession struct {
	ssh.Channel
	remoteAddr string
	user       string
	IsPty       bool
	PtyWidth    int
	PtyHeight   int
	Env         map[string]string
	closed      sync.Once
}

func (s *SSHSession) Close() error {
	s.closed.Do(func() {
		s.Channel.Close()
	})
	return nil
}

func (s *SSHSession) RemoteAddr() string { return s.remoteAddr }
func (s *SSHSession) User() string       { return s.user }

type SSHServerConfig struct {
	Addr           string
	HostKey        []byte
	NewProgram     func(*SSHSession) *Program
	Middlewares    []Middleware
	PasswordAuth   func(user, password string) bool
	PublicKeyAuth  func(key ssh.PublicKey) bool
	MaxSessions    int
	ReadTimeout    time.Duration
	Logger         *log.Logger
}

type SSHServer struct {
	sshConfig   *ssh.ServerConfig
	listener    net.Listener
	newProgram  func(*SSHSession) *Program
	middleware  SessionHandler
	sessions    int64
	ctx         context.Context
	cancel      context.CancelFunc
	logger      *log.Logger
	maxSessions int
	conns       map[string]net.Conn
	connsMu     sync.Mutex
}

func NewSSHServer(cfg SSHServerConfig) (*SSHServer, error) {
	sshCfg := &ssh.ServerConfig{}

	if cfg.PasswordAuth != nil {
		sshCfg.PasswordCallback = func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if cfg.PasswordAuth(conn.User(), string(password)) {
				return &ssh.Permissions{
					Extensions: map[string]string{"user": conn.User()},
				}, nil
			}
			return nil, fmt.Errorf("authentication failed")
		}
	}

	if cfg.PublicKeyAuth != nil {
		sshCfg.PublicKeyCallback = func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if cfg.PublicKeyAuth(key) {
				return &ssh.Permissions{
					Extensions: map[string]string{
						"user":       conn.User(),
						"pubkey":     string(key.Marshal()),
						"fingerprint": ssh.FingerprintSHA256(key),
					},
				}, nil
			}
			return nil, fmt.Errorf("authentication failed")
		}
	}

	if cfg.HostKey != nil {
		signer, err := ssh.ParsePrivateKey(cfg.HostKey)
		if err != nil {
			return nil, fmt.Errorf("ssh: invalid host key: %w", err)
		}
		sshCfg.AddHostKey(signer)
	} else {
		signer, err := generateOrLoadHostKey()
		if err != nil {
			return nil, fmt.Errorf("ssh: host key: %w", err)
		}
		sshCfg.AddHostKey(signer)
	}

	if cfg.NewProgram == nil {
		return nil, fmt.Errorf("ssh: NewProgram is required")
	}

	var handler SessionHandler = defaultSessionHandler(cfg.NewProgram)
	for i := len(cfg.Middlewares) - 1; i >= 0; i-- {
		handler = cfg.Middlewares[i](handler)
	}

	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}

	maxSessions := cfg.MaxSessions
	if maxSessions <= 0 {
		maxSessions = 64
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &SSHServer{
		sshConfig:   sshCfg,
		newProgram:  cfg.NewProgram,
		middleware:  handler,
		ctx:         ctx,
		cancel:      cancel,
		logger:      logger,
		maxSessions: maxSessions,
		conns:       make(map[string]net.Conn),
	}, nil
}

func defaultSessionHandler(newProgram func(*SSHSession) *Program) SessionHandler {
	return func(sess *SSHSession) {
		p := newProgram(sess)
		_ = p.Run()
	}
}

func (s *SSHServer) Serve(addr string) error {
	var err error
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("ssh: listen: %w", err)
	}

	s.logger.Printf("ssh: listening on %s", addr)

	go s.acceptLoop()
	<-s.ctx.Done()
	return nil
}

func (s *SSHServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				s.logger.Printf("ssh: accept: %v", err)
				continue
			}
		}
		go s.handleConn(conn)
	}
}

func (s *SSHServer) handleConn(netConn net.Conn) {
	remoteAddr := netConn.RemoteAddr().String()

	s.connsMu.Lock()
	s.conns[remoteAddr] = netConn
	s.connsMu.Unlock()

	defer func() {
		s.connsMu.Lock()
		delete(s.conns, remoteAddr)
		s.connsMu.Unlock()
	}()

	if s.maxSessions > 0 && atomic.LoadInt64(&s.sessions) >= int64(s.maxSessions) {
		s.logger.Printf("ssh: max sessions reached, rejecting %s", remoteAddr)
		netConn.Close()
		return
	}

	sshConn, chans, reqs, err := ssh.NewServerConn(netConn, s.sshConfig)
	if err != nil {
		s.logger.Printf("ssh: handshake failed from %s: %v", remoteAddr, err)
		netConn.Close()
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	user := ""
	if perms := sshConn.Permissions; perms != nil {
		user = perms.Extensions["user"]
	}

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}
		go s.handleChannel(newChan, remoteAddr, user)
	}
}

func (s *SSHServer) handleChannel(newChan ssh.NewChannel, remoteAddr, user string) {
	ch, chReqs, err := newChan.Accept()
	if err != nil {
		s.logger.Printf("ssh: channel accept: %v", err)
		return
	}

	atomic.AddInt64(&s.sessions, 1)
	defer atomic.AddInt64(&s.sessions, -1)

	sess := &SSHSession{
		Channel:    ch,
		remoteAddr: remoteAddr,
		user:       user,
		Env:        make(map[string]string),
	}

	go func() {
		for req := range chReqs {
			switch req.Type {
			case "pty-req":
				sess.IsPty = true
				s.parsePtyReq(req.Payload)
				if req.WantReply {
					req.Reply(true, nil)
				}
			case "window-change":
				s.parseWindowChange(req.Payload, sess)
				if req.WantReply {
					req.Reply(true, nil)
				}
			case "env":
				s.parseEnvReq(req.Payload, sess)
				if req.WantReply {
					req.Reply(true, nil)
				}
			case "exec":
				if req.WantReply {
					req.Reply(true, nil)
				}
			case "subsystem":
				if req.WantReply {
					req.Reply(false, nil)
				}
			default:
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}
	}()

	s.middleware(sess)
}

func (s *SSHServer) parsePtyReq(payload []byte) {
	if len(payload) < 4 {
		return
	}
	termLen := int(payload[3])
	if len(payload) < 4+termLen+8 {
		return
	}
	offset := 4 + termLen
	width := int(payload[offset])<<8 | int(payload[offset+1])
	height := int(payload[offset+2])<<8 | int(payload[offset+3])
	s.logger.Printf("ssh: pty-req: %dx%d", width, height)
	_ = width
	_ = height
}

func (s *SSHServer) parseWindowChange(payload []byte, sess *SSHSession) {
	if len(payload) < 8 {
		return
	}
	sess.PtyWidth = int(payload[0])<<8 | int(payload[1])
	sess.PtyHeight = int(payload[2])<<8 | int(payload[3])
}

func (s *SSHServer) parseEnvReq(payload []byte, sess *SSHSession) {
	if len(payload) < 8 {
		return
	}
	nameLen := int(payload[3])
	if len(payload) < 4+nameLen+4 {
		return
	}
	name := string(payload[4 : 4+nameLen])
	offset := 4 + nameLen
	valLen := int(payload[offset])<<8 | int(payload[offset+1])
	offset += 2
	if len(payload) < offset+valLen {
		return
	}
	val := string(payload[offset : offset+valLen])
	sess.Env[name] = val
}

func (s *SSHServer) Sessions() int64 {
	return atomic.LoadInt64(&s.sessions)
}

func (s *SSHServer) Close() error {
	s.cancel()
	s.connsMu.Lock()
	for addr, conn := range s.conns {
		conn.Close()
		delete(s.conns, addr)
	}
	s.connsMu.Unlock()
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *SSHServer) Addr() net.Addr {
	if s.listener != nil {
		return s.listener.Addr()
	}
	return nil
}

func generateOrLoadHostKey() (ssh.Signer, error) {
	keyPath := os.Getenv("MOFU_SSH_KEY")
	if keyPath != "" {
		data, err := os.ReadFile(keyPath)
		if err == nil {
			return ssh.ParsePrivateKey(data)
		}
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}

	return signer, nil
}

func GenerateEd25519Key() ([]byte, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	privBytes, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(privBytes), nil
}

func LoggingMiddleware(logger *log.Logger) Middleware {
	if logger == nil {
		logger = log.Default()
	}
	return func(next SessionHandler) SessionHandler {
		return func(sess *SSHSession) {
			start := time.Now()
			logger.Printf("session opened: user=%s addr=%s pty=%v", sess.User(), sess.RemoteAddr(), sess.IsPty)
			next(sess)
			logger.Printf("session closed: user=%s addr=%s duration=%v", sess.User(), sess.RemoteAddr(), time.Since(start))
		}
	}
}

func RateLimitMiddleware(maxConcurrent int) Middleware {
	var count int64
	return func(next SessionHandler) SessionHandler {
		return func(sess *SSHSession) {
			if atomic.AddInt64(&count, 1) > int64(maxConcurrent) {
				atomic.AddInt64(&count, -1)
				fmt.Fprintf(sess, "\r\nServer full. Try again later.\r\n")
				sess.Close()
				return
			}
			defer atomic.AddInt64(&count, -1)
			next(sess)
		}
	}
}

func AuthMiddleware(callback func(user string, key ssh.PublicKey) bool) Middleware {
	return func(next SessionHandler) SessionHandler {
		return func(sess *SSHSession) {
			fmt.Fprintf(sess, "Authentication required.\r\n")
			next(sess)
		}
	}
}

func ContextMiddleware(ctx context.Context) Middleware {
	return func(next SessionHandler) SessionHandler {
		return func(sess *SSHSession) {
			go func() {
				<-ctx.Done()
				sess.Close()
			}()
			next(sess)
		}
	}
}

func PanicMiddleware(logger *log.Logger) Middleware {
	if logger == nil {
		logger = log.Default()
	}
	return func(next SessionHandler) SessionHandler {
		return func(sess *SSHSession) {
			defer func() {
				if r := recover(); r != nil {
					logger.Printf("session panic: user=%s addr=%s err=%v", sess.User(), sess.RemoteAddr(), r)
					fmt.Fprintf(sess, "\r\nInternal error.\r\n")
				}
			}()
			next(sess)
		}
	}
}

func ConcurrentLimitMiddleware(max int) Middleware {
	var count int64
	return func(next SessionHandler) SessionHandler {
		return func(sess *SSHSession) {
			current := atomic.AddInt64(&count, 1)
			defer atomic.AddInt64(&count, -1)
			if current > int64(max) {
				fmt.Fprintf(sess, "\r\nToo many sessions. Limit: %d\r\n", max)
				return
			}
			next(sess)
		}
	}
}
