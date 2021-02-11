package main

import (
	"PortForwardGo/zlog"
	"net"
	"time"

	proxyprotocol "github.com/pires/go-proxyproto"
)

func LoadTCPRules(i string) {
	Setting.mu.Lock()
	tcpaddress, _ := net.ResolveTCPAddr("tcp", ":"+Setting.Config.Rules[i].Listen)
	ln, err := net.ListenTCP("tcp", tcpaddress)
	if err == nil {
		zlog.Info("Loaded [", i, "] (TCP)", Setting.Config.Rules[i].Listen, " => ", Setting.Config.Rules[i].Forward)
	} else {
		zlog.Error("Load failed [", i, "] (TCP) Error: ", err)
		SendListenError(i)
		Setting.mu.Unlock()
		return
	}
	Setting.Listener.TCP[i] = ln
	Setting.mu.Unlock()
	for {
		conn, err := ln.Accept()

		if err != nil {
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			break
		}

		go func() {
			Setting.mu.RLock()
			rule := Setting.Config.Rules[i]
			Setting.mu.RUnlock()

			if rule.Status != "Active" && rule.Status != "Created" {
				conn.Close()
				return
			}

			go tcp_handleRequest(conn, i, rule)
		}()
	}
}

func DeleteTCPRules(i string) {
	if _, ok := Setting.Listener.TCP[i]; ok {
		err := Setting.Listener.TCP[i].Close()
		for err != nil {
			time.Sleep(time.Second)
			err = Setting.Listener.TCP[i].Close()
		}
		delete(Setting.Listener.TCP, i)
	}
	Setting.mu.Lock()
	zlog.Info("Deleted [", i, "] (TCP)", Setting.Config.Rules[i].Listen, " => ", Setting.Config.Rules[i].Forward)
	delete(Setting.Config.Rules, i)
	Setting.mu.Unlock()
}

func tcp_handleRequest(conn net.Conn, index string, r Rule) {
	proxy, err := net.Dial("tcp", r.Forward)
	if err != nil {
		conn.Close()
		return
	}

	if r.ProxyProtocolVersion != 0 {
		header, err := proxyprotocol.HeaderProxyFromAddrs(byte(r.ProxyProtocolVersion), conn.RemoteAddr(), conn.LocalAddr()).Format()
		if err == nil {
			limitWrite(proxy, r.UserID, header)
		}
	}

	go copyIO(conn, proxy, r.UserID)
	go copyIO(proxy, conn, r.UserID)
}
