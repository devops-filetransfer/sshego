package sshego

import (
	"net"
	"strings"

	"golang.org/x/crypto/ssh"
)

// DialConfig provides Dial() with what
// it needs in order to establish an encrypted
// and authenticated ssh connection.
//
type DialConfig struct {

	// ClientKnownHostsPath is the path to the file
	// on client's disk that holds the known server keys.
	ClientKnownHostsPath string

	// cached to avoid a disk read, we only read
	// from ClientKnownHostsPath if KnownHosts is nil.
	// Users of DialConfig can leave this nil and
	// simply provide ClientKnownHostsPath. It is
	// exposed in case you need to invalidate the
	// cache and start again.
	KnownHosts *KnownHosts

	// the username to login under
	Mylogin string

	// the path on the local file system (client side) from
	// which to read the client's RSA private key.
	RsaPath string

	// the time-based one-time password configuration
	Totp string

	// Pw is the passphrase
	Pw string

	// which sshd to connect to, host and port.
	Sshdhost string
	Sshdport uint64

	// DownstreamHostPort is the host:port string of
	// the tcp address to which the sshd should forward
	// our connection to.
	DownstreamHostPort string

	// TofuAddIfNotKnown, for maximum security,
	// should be left false and
	// the host key database should be configured
	// manually. If true, the client trusts the server's
	// provided key and stores it, which creates
	// vulnerability to a MITM attack.
	//
	// TOFU stands for Trust-On-First-Use.
	//
	// If set to true, Dial() will set TofuAddIfNotKnown back
	// to false after storing the server (or
	// attacker) provided key and retying the
	// connection attempt with the newly stored
	// key. This prevents MITM after the
	// first contact if the DialConfig is reused.
	TofuAddIfNotKnown bool
}

// Dial is a convenience method for contacting an sshd
// over tcp and creating a direct-tcpip encrypted stream.
// It is a simple two-step sequence of calling
// dc.Cfg.SSHConnect() and then calling Dial() on the
// returned *ssh.Client.
//
// PRE: dc.Cfg.KnownHosts should already be instantiated.
// To prevent MITM attacks, the host we contact at
// hostport must have its server key must be already
// in the KnownHosts.
//
// dc.RsaPath is the path to the our (the client's) rsa
// private key file.
//
// dc.DownstreamHostPort is the host:port tcp address string
// to which the sshd should forward our connection after successful
// authentication.
//
func (dc *DialConfig) Dial() (net.Conn, *ssh.Client, error) {

	cfg := NewSshegoConfig()
	cfg.BitLenRSAkeys = 4096
	cfg.DirectTcp = true
	cfg.AddIfNotKnown = dc.TofuAddIfNotKnown
	if dc.KnownHosts == nil {
		dc.KnownHosts = NewKnownHosts(dc.ClientKnownHostsPath)
	}

	tryCount := 1
	if dc.TofuAddIfNotKnown {
		// need 2nd pass to actually connect; first pass
		// we cache the server key, 2nd time we actually connect.
		tryCount = 2
	}

	var sshClientConn *ssh.Client
	var err error
	for i := 0; i < tryCount; i++ {
		sshClientConn, err = cfg.SSHConnect(dc.KnownHosts,
			dc.Mylogin, dc.RsaPath, dc.Sshdhost, dc.Sshdport, dc.Pw, dc.Totp)
		//pp("sshClientConn = %#v", sshClientConn)
		if err != nil && strings.Contains(err.Error(), "Re-run without -new now") {
			if cfg.AddIfNotKnown {
				cfg.AddIfNotKnown = false
				dc.TofuAddIfNotKnown = false
				continue
			}
			return nil, nil, err
		}
	}
	if err != nil {
		return nil, nil, err
	}
	// Here is how to dial over an encrypted ssh channel.
	// This produces direct-tcpip forwarding -- in other
	// words we talk to the server at dest via the sshd,
	// but no other port is opened and so we have
	// exclusive access. This prevents other users and
	// their processes on this localhost from also
	// using the ssh connection (i.e. without authenticating).
	nc, err := sshClientConn.Dial("tcp", dc.DownstreamHostPort)
	return nc, sshClientConn, err
}
