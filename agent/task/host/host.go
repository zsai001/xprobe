package host

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
	"xprobe_agent/log"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type ServerStaticData struct {
	ID             string `json:"id"`
	HostName       string `json:"hostName"`
	OSName         string `json:"osName"`
	NAT            bool   `json:"nat"`
	OSVersion      string `json:"osVersion"`
	Architecture   string `json:"osArchitecture"`
	Virtualization string `json:"virtualization"`
	PublicIPV4     string `json:"publicIPV4"`
	PublicIPV6     string `json:"publicIPV6"`
	Isp            string `json:"isp"`
	VendorName     string `json:"vendorName"`
	CountryCode    string `json:"countryCode"`
	IPv4Supported  bool   `json:"ipv4Supported"`
	IPv6Supported  bool   `json:"ipv6Supported"`
	SwapTotal      string `json:"swapTotal"`
	MemoryTotal    string `json:"memoryTotal"`
	DiskTotal      string `json:"diskTotal"`
	UpDateTime     string `json:"upDateTime"`
	Version        string `json:"version"`
}

type HostInfoConfig struct {
	NodeID  string
	Version string
}

type HostInfoNode struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type HostInfoTask struct {
	Config *HostInfoConfig
	Result []ServerStaticData
}

func (t *HostInfoTask) GetData() interface{} {
	data := t.Result
	t.Result = t.Result[:]
	return data
}

func (t *HostInfoTask) SetConfig(cfg string) string {
	t.Config = &HostInfoConfig{}
	json.Unmarshal([]byte(cfg), t.Config)
	return ""
}

func (t *HostInfoTask) Execute() error {
	data := ServerStaticData{}
	hostInfo, err := host.Info()
	if err != nil {
		return fmt.Errorf("获取主机信息失败: %v", err)
	}
	data.HostName = hostInfo.Hostname
	data.OSName = hostInfo.OS
	data.OSVersion = hostInfo.PlatformVersion
	data.Architecture = hostInfo.KernelArch
	data.Virtualization = hostInfo.VirtualizationSystem
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("获取内存信息失败: %v", err)
	}
	data.MemoryTotal = fmt.Sprint(memInfo.Total)
	diskInfo, err := getDiskInfo()
	if err != nil {
		return fmt.Errorf("获取磁盘信息失败: %v", err)
	}
	data.DiskTotal = fmt.Sprint(diskInfo.Total)
	ipv4Supported, ipv6Supported := checkIPSupport()
	data.IPv4Supported = ipv4Supported
	data.IPv6Supported = ipv6Supported
	publicIpv4, publicIpv6, err := getPublicIP()
	if err != nil {
		publicIpv4 = "Unknown"
		publicIpv6 = "Unknown"
	}
	data.PublicIPV4 = publicIpv4
	data.PublicIPV6 = publicIpv6
	_, ipv4, _ := getLocalIPs()
	isNAT := checkNAT(publicIpv4, ipv4)
	data.NAT = isNAT
	data.Isp = "Unknown"
	data.VendorName = "Unknown"
	data.CountryCode = "Unknown"
	data.UpDateTime = time.Now().Format(time.RFC3339)
	t.Result = append(t.Result, data)
	if len(t.Result) > 10 {
		t.Result = t.Result[:10]
	}
	log.Info("run host task with", data)
	return nil
}

func getLocalIPs() (string, string, string) {
	var ips, ipv4, ipv6 []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", "", ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipv4 = append(ipv4, ipnet.IP.String())
			} else {
				ipv6 = append(ipv6, ipnet.IP.String())
			}
			ips = append(ips, ipnet.IP.String())
		}
	}

	return strings.Join(ips, ","), strings.Join(ipv4, ","), strings.Join(ipv6, ",")
}

func checkNAT(publicIP, localIPs string) bool {
	if publicIP == "Unknown" || localIPs == "" {
		return false // 无法确定，默认为非 NAT
	}

	localIPList := strings.Split(localIPs, ",")
	for _, localIP := range localIPList {
		if localIP == publicIP {
			return false // 公网 IP 与某个本地 IP 匹配，不是 NAT
		}
	}
	return true // 公网 IP 与所有本地 IP 都不匹配，可能是 NAT
}

func getDiskInfo() (struct{ Used, Total uint64 }, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return struct{ Used, Total uint64 }{}, err
	}

	var totalUsed, totalSpace uint64

	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}
		totalUsed += usage.Used
		totalSpace += usage.Total
	}

	return struct{ Used, Total uint64 }{totalUsed, totalSpace}, nil
}

func checkIPSupport() (bool, bool) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false, false
	}

	ipv4Supported := false
	ipv6Supported := false

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipv4Supported = true
			} else {
				ipv6Supported = true
			}
		}
	}

	return ipv4Supported, ipv6Supported
}

func getPublicIP() (string, string, error) {
	// 这里需要实现获取公网IP的逻辑
	// 可以使用第三方服务或者其他方法
	return "Unknown", "Unknown", nil
}
