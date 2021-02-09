package main

import (
	"net"
	"sync"
	"time"

	"github.com/CoiaPrant/PortForwardGo/zlog"
	proxyprotocol "github.com/pires/go-proxyproto"
)

const MTU = 65535

type UDPDistribute struct {
	Conn  *(net.UDPConn)
	RAddr net.Addr
	LAddr net.Addr
}

func NewUDPDistribute(conn *(net.UDPConn), addr net.Addr) *UDPDistribute {
	return &UDPDistribute{
		Conn:  conn,
		RAddr: addr,
		LAddr: conn.LocalAddr(),
	}
}

func (this *UDPDistribute) Close() error {
	return this.Conn.Close()
}

func (this *UDPDistribute) Read(b []byte) (n int, err error) {
	n, _, err = this.Conn.ReadFrom(b)
	return
}

func (this *UDPDistribute) Write(b []byte) (n int, err error) {
	return this.Conn.WriteTo(b, this.RAddr)
}

func (this *UDPDistribute) RemoteAddr() net.Addr {
	return this.RAddr
}

func (this *UDPDistribute) LocalAddr() net.Addr {
	return this.LAddr
}

func (this *UDPDistribute) SetDeadline(t time.Time) error {
	return this.Conn.SetDeadline(t)
}

func (this *UDPDistribute) SetReadDeadline(t time.Time) error {
	return this.Conn.SetReadDeadline(t)
}

func (this *UDPDistribute) SetWriteDeadline(t time.Time) error {
	return this.Conn.SetWriteDeadline(t)
}

func LoadUDPRules(i string) {

	Setting.mu.Lock()
	address, _ := net.ResolveUDPAddr("udp", ":"+Setting.Config.Rules[i].Listen)
	ln, err := net.ListenUDP("udp", address)
	if err == nil {
		zlog.Info("Loaded [", i, "] (UDP)", Setting.Config.Rules[i].Listen, " => ", Setting.Config.Rules[i].Forward)
	} else {
		zlog.Error("Load failed [", i, "] (UDP) Error: ", err)
		SendListenError(i)
		Setting.mu.Unlock()
		return
	}

	Setting.Listener.UDP[i] = ln
	Setting.mu.Unlock()

	go AcceptUDP(ln, i)
}

func DeleteUDPRules(i string) {
	if _, ok := Setting.Listener.UDP[i]; ok {
		err := Setting.Listener.UDP[i].Close()
		for err != nil {
			time.Sleep(time.Second)
			err = Setting.Listener.UDP[i].Close()
		}
		delete(Setting.Listener.UDP, i)
	}
	Setting.mu.Lock()
	delete(Setting.Config.Rules, i)
	Setting.mu.Unlock()
}

func UDPRelay(src, dst *net.UDPConn, writeTo net.Addr, index string) {
	// handle coping by hand to keep connection dynamically
	d := NewUDPDistribute(dst, writeTo)

	// not support jumbo frame
	buf := make([]byte, MTUTrie.GetMTU(writeTo.(*net.UDPAddr).IP))

	for {
		src.SetReadDeadline(time.Now().Add(2 * time.Minute))
		n, err := src.Read(buf)
		if err != nil {
			return
		}
		_, err = limitWrite(d, index, buf[:n])
		if err != nil {
			return
		}
	}
}

func AcceptUDP(serv *net.UDPConn, index string) {

	table := sync.Map{}

	buf := make([]byte, MTU)

	for {
		n, addr, err := serv.ReadFrom(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			break
		}

		Setting.mu.RLock()
		rule := Setting.Config.Rules[index]
		Setting.mu.RUnlock()

		if rule.Status != "Active" && rule.Status != "Created" {
			continue
		}

		if d, ok := table.Load(addr.String()); ok {
			go d.(*net.UDPConn).Write(buf[:n])
			continue
		}

		proxy, err := DialTarget(rule, addr, serv.LocalAddr())
		if err != nil {
			continue
		}
		table.Store(addr.String(), proxy)

		go func(addr net.Addr) {
			UDPRelay(proxy.(*net.UDPConn), serv, addr, index)
			table.Delete(addr.String())
			proxy.Close()
		}(addr)
		go proxy.Write(buf[:n])
	}
}
func DialTarget(r Rule, srcRAddr, srcLAddr net.Addr) (proxy net.Conn, err error) {
	proxy, err = ConnUDP(r.Forward)
	if err != nil {
		return
	}
	if r.ProxyProtocolVersion != 0 {
		header := proxyprotocol.HeaderProxyFromAddrs(byte(r.ProxyProtocolVersion), srcRAddr, srcLAddr)
		header.WriteTo(proxy)
	}
	return
}
func ConnUDP(address string) (net.Conn, error) {
	conn, err := net.DialTimeout("udp", address, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
