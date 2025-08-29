package convert

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/xmdhs/clash2singbox/model"
	"github.com/xmdhs/clash2singbox/model/clash"
	"github.com/xmdhs/clash2singbox/model/singbox"
)

var convertMap = map[string]func(*clash.Proxies, *singbox.SingBoxOut, model.SingBoxVer) ([]singbox.SingBoxOut, error){
	"vmess":       oldConver(vmess),
	"vless":       oldConver(vless),
	"shadowsocks": ss,
	// "shadowsocksr": ss,
	"trojan":    oldConver(trojan),
	"http":      oldConver(httpOpts),
	"socks":     oldConver(socks5),
	"hysteria":  oldConver(hysteria),
	"hysteria2": hysteia2,
	"wireguard": wireguard,
	"tuic":      tuic,
	"anytls":    anytls,
}

func oldConver(f func(*clash.Proxies, *singbox.SingBoxOut) error) func(*clash.Proxies, *singbox.SingBoxOut, model.SingBoxVer) ([]singbox.SingBoxOut, error) {
	return func(c *clash.Proxies, p *singbox.SingBoxOut, _ model.SingBoxVer) ([]singbox.SingBoxOut, error) {
		err := f(c, p)
		return []singbox.SingBoxOut{*p}, err
	}
}

func Clash2sing(c clash.Clash, ver model.SingBoxVer) ([]singbox.SingBoxOut, error) {
	sl := make([]singbox.SingBoxOut, 0, len(c.Proxies)+1)
	var jerr error
	for _, v := range c.Proxies {
		v := v
		s, t, err := comm(&v)
		if err != nil {
			jerr = errors.Join(jerr, err)
			continue
		}
		nsl, err := convertMap[t](&v, s, ver)
		if err != nil {
			jerr = errors.Join(jerr, err)
			continue
		}
		sl = append(sl, nsl...)
	}
	slm := make(map[string]singbox.SingBoxOut, len(c.Proxies)+1)
	for _, v := range sl {
		slm[v.Tag] = v
	}
	for _, v := range c.ProxyGroup {
		if v.Type != "relay" {
			continue
		}
		l := relay(slm, v.Proxies, v.Name)
		sl = append(sl, l...)
	}

	return sl, jerr
}

var ErrNotSupportType = errors.New("不支持的类型")

var typeMap = map[string]string{
	"ss": "shadowsocks",
	// "ssr":       "shadowsocksr",
	"vmess":     "vmess",
	"vless":     "vless",
	"trojan":    "trojan",
	"socks5":    "socks",
	"http":      "http",
	"hysteria":  "hysteria",
	"hysteria2": "hysteria2",
	"wireguard": "wireguard",
	"tuic":      "tuic",
	"anytls":    "anytls",
}

func comm(p *clash.Proxies) (*singbox.SingBoxOut, string, error) {
	s := &singbox.SingBoxOut{}
	s.Type = typeMap[p.Type]
	if s.Type == "" {
		return nil, "", fmt.Errorf("comm: %w %v", ErrNotSupportType, p.Type)
	}
	s.Tag = p.Name
	s.Server = p.Server
	port, err := strconv.Atoi(p.Port)
	if err != nil {
		return nil, "", fmt.Errorf("comm: %w", err)
	}
	s.ServerPort = port
	s.Password = p.Password

	if p.Smux.Enabled {
		s.Multiplex = &singbox.SingMultiplex{
			Enabled:    true,
			MaxStreams: int(p.Smux.MaxStreams),
			Padding:    bool(p.Smux.Padding),
			Protocol:   p.Smux.Protocol,
		}
		if p.Smux.MaxStreams == 0 {
			s.Multiplex.MinStreams = max(int(p.Smux.MinStreams), 4)
			s.Multiplex.MaxConnections = max(int(p.Smux.MaxConnections), 4)
		}
	}
	s.TcpFastOpen = p.Tfo
	s.TcpMultiPath = p.Mptcp

	// 处理 network 字段
	if p.Network != "" {
		s.Network = p.Network
	}
	// 默认 network 由具体的协议处理函数设置

	return s, s.Type, nil
}
