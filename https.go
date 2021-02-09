package main

import (
	"PortForwardGo/zlog"
	"net"
	"strings"

	proxyprotocol "github.com/pires/go-proxyproto"
)

var https_index map[string]string

func HttpsInit() {
	https_index = make(map[string]string)
	zlog.Info("[HTTPS] Listening ", Setting.Config.Listen["Https"].Port)
	l, err := net.Listen("tcp", ":"+Setting.Config.Listen["Https"].Port)
	if err != nil {
		zlog.Error("[HTTPS] Listen failed , Error: ", err)
		return
	}
	for {
		c, err := l.Accept()
		if err != nil {
			continue
		}
		go https_handle(c)
	}
}

func LoadHttpsRules(i string) {
	Setting.mu.RLock()
	zlog.Info("Loaded [", i, "] (HTTPS)", Setting.Config.Rules[i].Listen, " => ", Setting.Config.Rules[i].Forward)
	https_index[strings.ToLower(Setting.Config.Rules[i].Listen)] = i
	Setting.mu.RUnlock()
}

func DeleteHttpsRules(i string) {
	Setting.mu.Lock()
	zlog.Info("Deleted [", i, "] (HTTPS)", Setting.Config.Rules[i].Listen, " => ", Setting.Config.Rules[i].Forward)
	delete(https_index, strings.ToLower(Setting.Config.Rules[i].Listen))
	delete(Setting.Config.Rules, i)
	Setting.mu.Unlock()
}

func https_handle(conn net.Conn) {
	firstByte := make([]byte, 1)
	_, error := conn.Read(firstByte)
	if error != nil {
		conn.Close()
		return
	}
	if firstByte[0] != 0x16 {
		conn.Close()
		return
	}

	versionBytes := make([]byte, 2)
	_, error = conn.Read(versionBytes)
	if error != nil {
		conn.Close()
		return
	}
	if versionBytes[0] < 3 || (versionBytes[0] == 3 && versionBytes[1] < 1) {
		conn.Close()
		return
	}

	restLengthBytes := make([]byte, 2)
	_, error = conn.Read(restLengthBytes)
	if error != nil {
		conn.Close()
		return
	}
	restLength := (int(restLengthBytes[0]) << 8) + int(restLengthBytes[1])

	rest := make([]byte, restLength)
	_, error = conn.Read(rest)
	if error != nil {
		conn.Close()
		return
	}

	current := 0
	if len(rest) == 0 {
		conn.Close()
		return
	}
	handshakeType := rest[0]
	current += 1
	if handshakeType != 0x1 {
		conn.Close()
		return
	}

	current += 3
	current += 2
	current += 4 + 28
	sessionIDLength := int(rest[current])
	current += 1
	current += sessionIDLength

	cipherSuiteLength := (int(rest[current]) << 8) + int(rest[current+1])
	current += 2
	current += cipherSuiteLength

	compressionMethodLength := int(rest[current])
	current += 1
	current += compressionMethodLength

	if current > restLength {
		conn.Close()
		return
	}

	current += 2

	hostname := ""
	for current < restLength && hostname == "" {
		extensionType := (int(rest[current]) << 8) + int(rest[current+1])
		current += 2

		extensionDataLength := (int(rest[current]) << 8) + int(rest[current+1])
		current += 2

		if extensionType == 0 {
			current += 2

			nameType := rest[current]
			current += 1
			if nameType != 0 {
				conn.Close()
				return
			}
			nameLen := (int(rest[current]) << 8) + int(rest[current+1])
			current += 2
			hostname = strings.ToLower(string(rest[current : current+nameLen]))
		}

		current += extensionDataLength
	}

	if hostname == "" {
		conn.Close()
		return
	}

	i, ok := https_index[hostname]
	if !ok {
		conn.Close()
		return
	}

	Setting.mu.RLock()
	rule := Setting.Config.Rules[i]
	Setting.mu.RUnlock()

	if rule.Status != "Active" && rule.Status != "Created" {
		conn.Close()
		return
	}

	backend, error := net.Dial("tcp", rule.Forward)
	if error != nil {
		conn.Close()
		return
	}

	if rule.ProxyProtocolVersion != 0 {
		header := proxyprotocol.HeaderProxyFromAddrs(byte(rule.ProxyProtocolVersion), conn.RemoteAddr(), conn.LocalAddr())
		header.WriteTo(backend)
	}

	backend.Write(firstByte)
	backend.Write(versionBytes)
	backend.Write(restLengthBytes)
	backend.Write(rest)

	go copyIO(conn, backend, i)
	go copyIO(backend, conn, i)
}
