// verify-clash-profile emits a credential-free subscription fixture for an
// external Mihomo parser test. It does not connect to a database or panel server.
package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"trojan-panel/model/bo"
	"trojan-panel/model/constant"
	"trojan-panel/util"
)

func main() {
	target := flag.String("target", util.ClashVergeTarget, "subscription target; empty tests the legacy profile")
	flag.Parse()
	nodes := []interface{}{
		bo.Trojan{Name: "Trojan", Type: "trojan", Server: "example.com", Port: 443, Password: "test-only-not-a-secret", Udp: true},
		bo.Vless{Name: "VLESS", Type: "vless", Server: "example.com", Port: 443, Uuid: "11111111-1111-4111-8111-111111111111", Network: "tcp", Tls: true, Udp: true, ClientFingerprint: "chrome"},
		bo.Vmess{Name: "VMess", Type: "vmess", Server: "example.com", Port: 443, Uuid: "11111111-1111-4111-8111-111111111111", Network: "ws", Tls: true, Udp: true, Cipher: "auto", WsOpts: bo.WsOpts{Path: "/ws", Headers: bo.WsOptsHeaders{Host: "example.com"}}},
		bo.Shadowsocks{Name: "Shadowsocks", Type: "ss", Server: "example.com", Port: 443, Password: "test-only-not-a-secret", Cipher: "aes-128-gcm", Udp: true},
		bo.Socks{Name: "Socks", Type: "socks5", Server: "example.com", Port: 443, Udp: true},
		bo.TrojanGo{Name: "Trojan-Go", Type: "trojan", Server: "example.com", Port: 443, Password: "test-only-not-a-secret", Udp: true, Network: "ws", WsOpts: bo.WsOpts{Path: "/ws"}},
		bo.Hysteria{Name: "Hysteria", Type: "hysteria", Server: "example.com", Port: 443, AuthStr: "test-only-not-a-secret", Protocol: "udp", Up: 50, Down: 100},
		bo.Hysteria2{Name: "Hysteria2", Type: "hysteria2", Server: "example.com", Port: 443, Password: "test-only-not-a-secret", Up: 50, Down: 100},
	}
	names := []string{"Trojan", "VLESS", "VMess", "Shadowsocks", "Socks", "Trojan-Go", "Hysteria", "Hysteria2"}
	generated, err := yaml.Marshal(bo.ClashConfig{Proxies: nodes, ProxyGroups: []bo.ProxyGroup{{Name: "PROXY", Type: "select", Proxies: names}}})
	if err != nil {
		panic(err)
	}
	// Deliberately use the old persisted default, including the user's exact
	// failing rule, to verify migration rather than merely a fresh installation.
	old := strings.Replace(constant.ClashRules, "  - GEOIP,CN,DIRECT", "  - GEOIP,,DIRECT\n  - GEOIP,CN,DIRECT", 1)
	config, err := util.BuildClashProfile(generated, old, *target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(config)
}
