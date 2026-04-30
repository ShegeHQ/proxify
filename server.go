package main

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	whitelist, _ := os.LookupEnv("WHITELIST")
	allowed := allowedIPs(whitelist)

	httpStarted := false

	if target, ok := os.LookupEnv("TARGET"); ok {
		go startHTTPProxy(target, allowed)
		httpStarted = true
	}

	if redisTarget, ok := os.LookupEnv("REDIS_TARGET"); ok {
		redisTLS := strings.EqualFold(os.Getenv("REDIS_TLS"), "true")
		listen := os.Getenv("REDIS_LISTEN")
		if listen == "" {
			listen = ":6379"
		}
		go startTCPProxy(listen, redisTarget, redisTLS, allowed)
		httpStarted = true
	}

	if !httpStarted {
		log.Fatal("Set TARGET (HTTP) and/or REDIS_TARGET (TCP) in env")
	}

	select {}
}

func startHTTPProxy(target string, allowed []string) {
	remote, err := url.Parse(target)
	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = remote.Host
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !isAllowedIP(getClientIP(r), allowed) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		log.Printf("%s %s", r.Method, r.URL.String())
		proxy.ServeHTTP(w, r)
	})

	log.Println("Proxify HTTP running on :8888 →", target)
	if err := http.ListenAndServe(":8888", mux); err != nil {
		log.Fatal(err)
	}
}

func startTCPProxy(listen, target string, useTLS bool, allowed []string) {
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Proxify TCP running on %s → %s (tls=%v)", listen, target, useTLS)

	for {
		client, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handleTCP(client, target, useTLS, allowed)
	}
}

func handleTCP(client net.Conn, target string, useTLS bool, allowed []string) {
	defer client.Close()

	clientIP, _, err := net.SplitHostPort(client.RemoteAddr().String())
	if err != nil {
		return
	}
	if !isAllowedIP(clientIP, allowed) {
		log.Printf("TCP rejected: %s", clientIP)
		return
	}
	log.Printf("TCP %s → %s", clientIP, target)

	var upstream net.Conn
	if useTLS {
		host, _, _ := net.SplitHostPort(target)
		upstream, err = tls.Dial("tcp", target, &tls.Config{ServerName: host})
	} else {
		upstream, err = net.Dial("tcp", target)
	}
	if err != nil {
		log.Printf("dial upstream %s: %v", target, err)
		return
	}
	defer upstream.Close()

	done := make(chan struct{}, 2)
	go func() { io.Copy(upstream, client); done <- struct{}{} }()
	go func() { io.Copy(client, upstream); done <- struct{}{} }()
	<-done
}

func allowedIPs(whitelist string) []string {
	var ips []string
	for _, ip := range strings.Split(whitelist, ",") {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			ips = append(ips, ip)
		}
	}
	return ips
}

func isAllowedIP(ip string, allowedIPs []string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, cidr := range allowedIPs {
		if strings.Contains(cidr, "/") {
			_, subnet, err := net.ParseCIDR(cidr)
			if err == nil && subnet.Contains(parsedIP) {
				return true
			}
		} else {
			if cidr == ip {
				return true
			}
		}
	}

	return false
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
