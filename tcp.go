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
		zlog.Info("Loaded [",i,"] (TCP)", Setting.Config.Rules[i].Listen, " => ", Setting.Config.Rules[i].Forward)
	}else{
		zlog.Error("Load failed [",i,"] (TCP) Error: ",err)
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
		
		Setting.mu.RLock()
		rule := Setting.Config.Rules[i]

		if rule.Status != "Active" && rule.Status != "Created" {
			Setting.mu.RUnlock()
			conn.Close()
			continue
		}

		if Setting.Config.Users[rule.UserID].Used > Setting.Config.Users[rule.UserID].Quota { 			
			Setting.mu.RUnlock()
			conn.Close()
			continue
		}

		Setting.mu.RUnlock()

		go tcp_handleRequest(conn, i, rule)
	}
}

func DeleteTCPRules(i string){
	if _,ok :=Setting.Listener.TCP[i];ok {
		err :=Setting.Listener.TCP[i].Close()
		for err!=nil {
		time.Sleep(time.Second)
		err = Setting.Listener.TCP[i].Close()
		}
	}
	Setting.mu.Lock()
	zlog.Info("Deleted [",i,"] (TCP)", Setting.Config.Rules[i].Listen, " => ", Setting.Config.Rules[i].Forward)
	delete(Setting.Config.Rules,i)
	delete(Setting.Listener.TCP,i)
	Setting.mu.Unlock()
}

func tcp_handleRequest(conn net.Conn, index string, r Rule) {
	proxy, err := net.Dial("tcp", r.Forward)
	if err != nil {
		_ = conn.Close()
		return
	}
	
	if r.ProxyProtocolVersion != 0 {
	header := proxyprotocol.HeaderProxyFromAddrs(byte(r.ProxyProtocolVersion),conn.RemoteAddr(),conn.LocalAddr())
	header.WriteTo(proxy)
	}
	
	go copyIO(conn, proxy, index)
	go copyIO(proxy, conn, index)
}