package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"code.google.com/p/go.net/publicsuffix"
	"github.com/hailiang/socks"
)

var (
	proxy      = flag.Bool("p", false, "是否通过代理访问，默认是")
	proxyStr   = flag.String("pu", "127.0.0.1:9150", `代理地址，格式:"SOCKS5://127.0.0.1:1080" or "127.0.0.1:9150"，默认是Tor 代理`)
	httpClient = &http.Client{}
)

func InitProxy() {
	proxyURL := strings.ToUpper(*proxyStr)
	fmt.Println(proxyURL)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		ResponseHeaderTimeout: time.Minute * 2,
	}
	jar, _ := cookiejar.New(
		&cookiejar.Options{
			publicsuffix.List,
		},
	)
	defer func() {
		httpClient = &http.Client{
			Jar:       jar,
			Transport: tr,
		}
	}()
	if *proxy {
		if !strings.HasPrefix(proxyURL, "HTTP") {
			//proxyURL like :"SOCKS5://127.0.0.1:1080" or "127.0.0.1:9150"
			switch proxyURL[:6] {
			case "SOCKS5":
				dialSocks5Proxy := socks.DialSocksProxy(socks.SOCKS5, proxyURL[9:])
				tr.Dial = dialSocks5Proxy
			case "SOCKS4":
				dialSocks4Proxy := socks.DialSocksProxy(socks.SOCKS4, proxyURL[9:])
				tr.Dial = dialSocks4Proxy
			default:
				proxyURL = strings.Replace(proxyURL, "SOCKS://", "", -1)
				dialSocksProxy := socks.DialSocksProxy(socks.SOCKS5, proxyURL)
				tr.Dial = dialSocksProxy
			}
		} else {
			pu, err := url.Parse(proxyURL)
			if err != nil {
				fmt.Println("InitProxy url.Parse:", err)
				return
			}
			tr.Proxy = http.ProxyURL(pu)
			tr.Dial = func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, time.Second*30)
				if err != nil {
					return nil, err
				}
				deadline := time.Now().Add(40 * time.Second)
				c.SetDeadline(deadline)
				return c, nil
			}
		}
	}
}
