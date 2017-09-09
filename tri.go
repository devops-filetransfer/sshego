package sshego

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	ssh "github.com/glycerine/sshego/xendor/github.com/glycerine/xcryptossh"
)

// Tricorder records (holds) three key objects:
//   an ssh.Channel, an *ssh.Client, and the underlyign net.Conn.
//
// Tricorder supports auto reconnect when disconnected.
//
type Tricorder struct {
	Halt *ssh.Halter // should only reflect close of the internal sshChannel, not cli nor nc.

	cfg *SshegoConfig

	cli         *ssh.Client
	nc          net.Conn
	uhp         *UHP
	sshChannels map[ssh.Channel]context.CancelFunc

	getChannelCh      chan *getChannelTicket
	getCliCh          chan *ssh.Client
	getNcCh           chan net.Conn
	reconnectNeededCh chan *UHP
}

func (cfg *SshegoConfig) NewTricorder(halt *ssh.Halter) (tri *Tricorder) {
	if halt == nil {
		halt = ssh.NewHalter()
	}
	tri = &Tricorder{
		cfg:  cfg,
		Halt: halt,

		sshChannels: make(map[ssh.Channel]context.CancelFunc),

		reconnectNeededCh: make(chan *UHP, 1),
		getChannelCh:      make(chan *getChannelTicket),
		getCliCh:          make(chan *ssh.Client),
		getNcCh:           make(chan net.Conn),
	}
	cfg.ClientReconnectNeededTower.Subscribe(tri.reconnectNeededCh)

	tri.startReconnectLoop()
	return tri
}

// CustomInprocStreamChanName is how sshego/reptile specific
// channels are named.
const CustomInprocStreamChanName = "custom-inproc-stream"

func (t *Tricorder) closeChannels() {
	if len(t.sshChannels) > 0 {
		for ch, cancel := range t.sshChannels {
			ch.Close()
			cancel()
		}
	}
	t.sshChannels = make(map[ssh.Channel]context.CancelFunc)
}

func (t *Tricorder) startReconnectLoop() {
	go func() {
		defer func() {
			t.closeChannels()
			t.Halt.MarkDone()
		}()
		for {
			select {
			case <-t.Halt.ReqStopChan():
				return
			case uhp := <-t.reconnectNeededCh:
				t.uhp = uhp
				t.closeChannels()
				t.cli = nil
				t.nc = nil
				// need to reconnect!
				t.helperNewClientConnect()

				// provide current state
			case t.getCliCh <- t.cli:
			case t.getNcCh <- t.nc:

				// bring up a new channel
			case tk := <-t.getChannelCh:
				t.helperGetChannel(tk)
			}
		}
	}()
}

func (t *Tricorder) helperNewClientConnect() {

	destHost, port, err := splitHostPort(t.uhp.HostPort)
	panicOn(err)

	ctx := context.Background()
	pw := ""
	toptUrl := ""
	//t.cfg.AddIfNotKnown = false
	sshcli, nc, err := t.cfg.SSHConnect(ctx, t.cfg.KnownHosts, t.uhp.User, t.cfg.PrivateKeyPath, destHost, int64(port), pw, toptUrl, t.Halt)
	if err != nil {
		panic(err)
	}
	t.cli = sshcli
	t.nc = nc
}

func (t *Tricorder) helperGetChannel(tk *getChannelTicket) {

	bkg := context.Background()
	ctx, cancelOpenChannelCtx := context.WithDeadline(bkg, time.Now().Add(5*time.Second))

	defer cancelOpenChannelCtx() // TODO: is this right??

	ch, in, err := t.cli.OpenChannel(ctx, CustomInprocStreamChanName, nil)
	if err == nil {

		discardCtx, discardCtxCancel := context.WithCancel(bkg)
		go DiscardRequestsExceptKeepalives(discardCtx, in, t.Halt.ReqStopChan())
		t.sshChannels[ch] = discardCtxCancel

		if ch != nil && t.cfg.IdleTimeoutDur > 0 {
			ch.SetIdleTimeout(t.cfg.IdleTimeoutDur)
		}
	}
	tk.sshChannel = ch
	tk.err = err

	close(tk.done)
}

type getChannelTicket struct {
	done       chan struct{}
	sshChannel ssh.Channel
	err        error
}

func newGetChannelTicket() *getChannelTicket {
	return &getChannelTicket{
		done: make(chan struct{}),
	}
}

func (t *Tricorder) SSHChannel() (ssh.Channel, error) {
	tk := newGetChannelTicket()
	t.getChannelCh <- tk
	<-tk.done
	return tk.sshChannel, tk.err
}

func (t *Tricorder) Cli() (cli *ssh.Client) {
	cli = <-t.getCliCh
	return
}

func (t *Tricorder) Nc() (nc net.Conn) {
	nc = <-t.getNcCh
	return
}

func splitHostPort(hostport string) (host string, port int, err error) {
	sPort := ""
	host, sPort, err = net.SplitHostPort(hostport)
	if err != nil {
		err = fmt.Errorf("bad addr '%s': net.SplitHostPort() gave: %s", hostport, err)
		return
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if len(sPort) == 0 {
		err = fmt.Errorf("no port found in '%s'", hostport)
		return
	}
	var prt uint64
	prt, err = strconv.ParseUint(sPort, 10, 16)
	if err != nil {
		return
	}
	port = int(prt)
	return
}
