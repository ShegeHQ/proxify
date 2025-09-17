package main

import (
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

	target, exist := os.LookupEnv("TARGET")

	if !exist {
		log.Fatal("TARGET not set in .env")
	}

	whitelist, _ := os.LookupEnv("WHITELIST")

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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !isAllowedIP(getClientIP(r), allowedIPs(whitelist)) {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		log.Printf("%s %s", r.Method, r.URL.String())
		proxy.ServeHTTP(w, r)
	})

	log.Println("Proxify running on :8888")

	if err := http.ListenAndServe(":8888", nil); err != nil {
		log.Fatal(err)
	}
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

	// Fallback: use RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
