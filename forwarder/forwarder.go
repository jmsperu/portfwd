package forwarder

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jmsperu/portfwd/stats"
)

type ForwardConfig struct {
	Name       string
	Protocol   string
	ListenPort int
	RemoteHost string
	RemotePort int
	TLSCert    string
	TLSKey     string
	AllowCIDRs []string
	DenyCIDRs  []string
	RateLimit  int
}

type Forwarder struct {
	cfg         ForwardConfig
	acl         *ACL
	tlsConfig   *tls.Config
	listener    net.Listener
	udpConn     *net.UDPConn
	stopCh      chan struct{}
	connections int64
	bytesIn     int64
	bytesOut    int64
	logger      *log.Logger
}

func New(cfg ForwardConfig) (*Forwarder, error) {
	acl, err := NewACL(cfg.AllowCIDRs, cfg.DenyCIDRs)
	if err != nil {
		return nil, fmt.Errorf("invalid ACL: %w", err)
	}

	f := &Forwarder{
		cfg:    cfg,
		acl:    acl,
		stopCh: make(chan struct{}),
		logger: log.New(os.Stdout, fmt.Sprintf("[%s] ", cfg.Name), log.LstdFlags),
	}

	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS cert/key: %w", err)
		}
		f.tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	return f, nil
}

func (f *Forwarder) Start() error {
	if f.cfg.Protocol == "udp" {
		return f.startUDP()
	}
	return f.startTCP()
}

func (f *Forwarder) Stop() {
	close(f.stopCh)
	if f.listener != nil {
		f.listener.Close()
	}
	if f.udpConn != nil {
		f.udpConn.Close()
	}
	f.recordStats()
}

func (f *Forwarder) recordStats() {
	stats.Record(stats.Entry{
		Name:        f.cfg.Name,
		Protocol:    f.cfg.Protocol,
		ListenPort:  f.cfg.ListenPort,
		RemoteHost:  f.cfg.RemoteHost,
		RemotePort:  f.cfg.RemotePort,
		Connections: atomic.LoadInt64(&f.connections),
		BytesIn:     atomic.LoadInt64(&f.bytesIn),
		BytesOut:    atomic.LoadInt64(&f.bytesOut),
		LastActive:  time.Now(),
	})
}

func (f *Forwarder) startTCP() error {
	addr := fmt.Sprintf(":%d", f.cfg.ListenPort)
	var err error

	if f.tlsConfig != nil {
		f.listener, err = tls.Listen("tcp", addr, f.tlsConfig)
	} else {
		f.listener, err = net.Listen("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			f.Stop()
		case <-f.stopCh:
		}
	}()

	for {
		conn, err := f.listener.Accept()
		if err != nil {
			select {
			case <-f.stopCh:
				return nil
			default:
				f.logger.Printf("Accept error: %v", err)
				continue
			}
		}

		if !f.acl.Allowed(conn.RemoteAddr()) {
			f.logger.Printf("Blocked connection from %s", conn.RemoteAddr())
			conn.Close()
			continue
		}

		atomic.AddInt64(&f.connections, 1)
		f.logger.Printf("New connection from %s (#%d)", conn.RemoteAddr(), atomic.LoadInt64(&f.connections))

		go f.handleTCPConn(conn)
	}
}

func (f *Forwarder) handleTCPConn(clientConn net.Conn) {
	defer clientConn.Close()

	remote := fmt.Sprintf("%s:%d", f.cfg.RemoteHost, f.cfg.RemotePort)
	remoteConn, err := net.DialTimeout("tcp", remote, 10*time.Second)
	if err != nil {
		f.logger.Printf("Failed to connect to %s: %v", remote, err)
		return
	}
	defer remoteConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Remote
	go func() {
		defer wg.Done()
		var r io.Reader = clientConn
		if f.cfg.RateLimit > 0 {
			r = NewRateLimitedReader(r, f.cfg.RateLimit)
		}
		n, _ := io.Copy(remoteConn, r)
		atomic.AddInt64(&f.bytesIn, n)
		// Signal the other direction to finish
		if tc, ok := remoteConn.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()

	// Remote -> Client
	go func() {
		defer wg.Done()
		var r io.Reader = remoteConn
		if f.cfg.RateLimit > 0 {
			r = NewRateLimitedReader(r, f.cfg.RateLimit)
		}
		n, _ := io.Copy(clientConn, r)
		atomic.AddInt64(&f.bytesOut, n)
		if tc, ok := clientConn.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()

	wg.Wait()
	f.logger.Printf("Connection from %s closed", clientConn.RemoteAddr())
}

func (f *Forwarder) startUDP() error {
	addr := fmt.Sprintf(":%d", f.cfg.ListenPort)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	f.udpConn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			f.Stop()
		case <-f.stopCh:
		}
	}()

	remote := fmt.Sprintf("%s:%d", f.cfg.RemoteHost, f.cfg.RemotePort)
	remoteAddr, err := net.ResolveUDPAddr("udp", remote)
	if err != nil {
		return fmt.Errorf("failed to resolve remote %s: %w", remote, err)
	}

	// Map of client addresses to their upstream connections
	type clientMapping struct {
		conn     *net.UDPConn
		lastSeen time.Time
	}
	clients := make(map[string]*clientMapping)
	var mu sync.Mutex

	// Cleanup stale clients every 60 seconds
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				for k, v := range clients {
					if time.Since(v.lastSeen) > 5*time.Minute {
						v.conn.Close()
						delete(clients, k)
					}
				}
				mu.Unlock()
			case <-f.stopCh:
				return
			}
		}
	}()

	buf := make([]byte, 65535)
	for {
		n, clientAddr, err := f.udpConn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-f.stopCh:
				return nil
			default:
				f.logger.Printf("UDP read error: %v", err)
				continue
			}
		}

		if !f.acl.Allowed(clientAddr) {
			f.logger.Printf("Blocked UDP from %s", clientAddr)
			continue
		}

		atomic.AddInt64(&f.bytesIn, int64(n))

		key := clientAddr.String()
		mu.Lock()
		mapping, exists := clients[key]
		if !exists {
			atomic.AddInt64(&f.connections, 1)
			f.logger.Printf("New UDP session from %s (#%d)", clientAddr, atomic.LoadInt64(&f.connections))

			upstreamConn, err := net.DialUDP("udp", nil, remoteAddr)
			if err != nil {
				f.logger.Printf("Failed to connect upstream: %v", err)
				mu.Unlock()
				continue
			}

			mapping = &clientMapping{conn: upstreamConn, lastSeen: time.Now()}
			clients[key] = mapping

			// Read responses from upstream and send back to client
			go func(cAddr *net.UDPAddr, upstream *net.UDPConn) {
				respBuf := make([]byte, 65535)
				for {
					upstream.SetReadDeadline(time.Now().Add(5 * time.Minute))
					rn, err := upstream.Read(respBuf)
					if err != nil {
						return
					}
					atomic.AddInt64(&f.bytesOut, int64(rn))
					f.udpConn.WriteToUDP(respBuf[:rn], cAddr)
				}
			}(clientAddr, upstreamConn)
		}
		mapping.lastSeen = time.Now()
		mu.Unlock()

		mapping.conn.Write(buf[:n])
	}
}
