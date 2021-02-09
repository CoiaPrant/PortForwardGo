package main

import (
	"PortForwardGo/zlog"
	"net"
	"time"

	"golang.org/x/net/websocket"
)

func LoadWSCRules(i string) {
	Setting.mu.Lock()
	tcpaddress, _ := net.ResolveTCPAddr("tcp", ":"+Setting.Config.Rules[i].Listen)

	ln, err := net.ListenTCP("tcp", tcpaddress)
	if err == nil {
		zlog.Info("Loaded [", i, "] (WebSocket Client) ", Setting.Config.Rules[i].Listen, " => ", Setting.Config.Rules[i].Forward)
	} else {
		zlog.Error("Load failed [", i, "] (WebSocket Client) Error:", err)
		Setting.mu.Unlock()
		SendListenError(i)
		return
	}

	Setting.Listener.WSC[i] = ln
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
		Setting.mu.RUnlock()

		if rule.Status != "Active" && rule.Status != "Created" {
			conn.Close()
			continue
		}

		go wsc_handleRequest(conn, i, rule)
	}
}

func DeleteWSCRules(i string) {
	if _, ok := Setting.Listener.WSC[i]; ok {
		err := Setting.Listener.WSC[i].Close()
		for err != nil {
			time.Sleep(time.Second)
			err = Setting.Listener.WSC[i].Close()
		}
		delete(Setting.Listener.WSC, i)
	}
	Setting.mu.Lock()
	zlog.Info("Deleted [", i, "] (WebSocket Client)", Setting.Config.Rules[i].Listen, " => ", Setting.Config.Rules[i].Forward)
	delete(Setting.Config.Rules, i)
	Setting.mu.Unlock()
}

func wsc_handleRequest(conn net.Conn, index string, r Rule) {
	ws_config, err := websocket.NewConfig("ws://"+r.Forward+"/ws/", "http://"+r.Forward+"/ws/")
	if err != nil {
		_ = conn.Close()
		return
	}
	ws_config.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36")
	ws_config.Header.Set("X-Forward-For", ParseAddrToIP(conn.RemoteAddr().String()))
	ws_config.Header.Set("X-Forward-Protocol", conn.RemoteAddr().Network())
	ws_config.Header.Set("X-Forward-Address", conn.RemoteAddr().String())
	proxy, err := websocket.DialConfig(ws_config)
	if err != nil {
		_ = conn.Close()
		return
	}
	proxy.PayloadType = websocket.BinaryFrame

	go copyIO(conn, proxy, index)
	go copyIO(proxy, conn, index)
}
