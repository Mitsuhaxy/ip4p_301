package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// Config 结构体用于解析 YAML 配置文件
type Config struct {
	Server struct {
		ListenPort  int    `yaml:"listen_port"`
		CertFile    string `yaml:"cert_file"`
		KeyFile     string `yaml:"key_file"`
	} `yaml:"server"`
	Mappings []struct {
		UUID   string `yaml:"uuid"`
		Domain string `yaml:"domain"`
	} `yaml:"mappings"`
}

// 读取并解析 YAML 配置文件
func loadConfig(filename string) (*Config, error) {
	config := &Config{}

	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	err = yaml.Unmarshal(file, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return config, nil
}

// 从 DNS 解析 AAAA 记录（IPv6 地址）
func resolveAAAA(domain string) (string, error) {
	ips, err := net.LookupHost(domain)
	if err != nil {
		return "", fmt.Errorf("failed to resolve domain: %v", err)
	}

	for _, ip := range ips {
		if strings.Contains(ip, ":") {
			return ip, nil
		}
	}
	return "", fmt.Errorf("no AAAA record found for domain: %s", domain)
}

func expandIPv6(ipv6 string) (string, error) {
		// 计算缺少的 0 的组数
		parts := strings.Split(ipv6, "::")
		left := strings.Split(parts[0], ":")
		right := strings.Split(parts[1], ":")
		missing := 8 - (len(left) + len(right))

		// 构造完整的 IPv6 地址
		expanded := parts[0] + ":"
		for i := 0; i < missing; i++ {
			expanded += "0:"
		}
		expanded += parts[1]
		return expanded, nil
}

// 解析 IP4P 地址，返回 IPv4 地址和端口号
func parseIP4P(ip4p string) (string, int, error) {
	// 先还原缩写的 IPv6 地址
	expanded, err := expandIPv6(ip4p)
	if err != nil {
		return "", 0, fmt.Errorf("failed to expand IPv6 address: %v", err)
	}

	// 拆分 IPv6 地址
	parts := strings.Split(expanded, ":")
	if len(parts) != 8 {
		return "", 0, fmt.Errorf("invalid IP4P address format")
	}

	// 提取 XXXX (端口号)
	xxxx := parts[5]
	port, err := strconv.ParseInt(xxxx, 16, 32)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse port: %v", err)
	}
	if port > 65535 {
		return "", 0, fmt.Errorf("invalid port number")
	}

	// 提取 YYYY (IPv4 高 16 位) 和 ZZZZ (IPv4 低 16 位)
	yyyy := parts[6]
	zzzz := parts[7]

	// 解析 YYYY 字段
	if len(yyyy) < 4 {
		for len(yyyy) < 4 {
			yyyy = "0" + yyyy
		}
	}
	ip1, err := strconv.ParseInt(yyyy[:2], 16, 32) // 第一段
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse IPv4 first segment: %v", err)
	}
	ip2, err := strconv.ParseInt(yyyy[2:], 16, 32) // 第二段
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse IPv4 second segment: %v", err)
	}

	// 解析 ZZZZ 字段
	if len(zzzz) < 4 {
		for len(zzzz) < 4 {
			zzzz = "0" + zzzz
		}
	}
	ip3, err := strconv.ParseInt(zzzz[:2], 16, 32) // 第三段
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse IPv4 third segment: %v", err)
	}
	ip4, err := strconv.ParseInt(zzzz[2:], 16, 32) // 第四段
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse IPv4 fourth segment: %v", err)
	}

	// 组合成完整的 IPv4 地址
	ip := fmt.Sprintf("%d.%d.%d.%d", ip1, ip2, ip3, ip4)
	log.Printf("Redirect https server to %s:%d \n", ip, port)
	return ip, int(port), nil
}

// 处理 HTTP 请求并重定向
func redirectHandler(config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uuid := strings.TrimPrefix(r.URL.Path, "/")
		if uuid == "" {
			http.Error(w, "UUID is required", http.StatusBadRequest)
			return
		}

		var targetDomain string
		for _, mapping := range config.Mappings {
			if mapping.UUID == uuid {
				targetDomain = mapping.Domain
				break
			}
		}
		if targetDomain == "" {
			http.Error(w, "UUID not found", http.StatusNotFound)
			return
		}

		ip4p, err := resolveAAAA(targetDomain)
		if err != nil {
			http.Error(w, "Failed to resolve AAAA record", http.StatusInternalServerError)
			return
		}

		ip, port, err := parseIP4P(ip4p)
		if err != nil {
			http.Error(w, "Failed to parse IP4P address", http.StatusInternalServerError)
			return
		}

		targetURL := fmt.Sprintf("https://%s:%d", ip, port)
		http.Redirect(w, r, targetURL, http.StatusMovedPermanently)
	}
}

func main() {
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	addr := fmt.Sprintf(":%d", config.Server.ListenPort)
	log.Printf("Starting https server on %s \n", addr)

	http.HandleFunc("/", redirectHandler(config))

	// 启动 HTTPS 服务器
	err = http.ListenAndServeTLS(
		addr,
		config.Server.CertFile,
		config.Server.KeyFile,
		nil,)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
