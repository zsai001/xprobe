package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

var version = "0.0.1"

type NetworkSpeed struct {
	Download uint64
	Upload   uint64
}

var (
	currentSpeed NetworkSpeed
	speedMutex   sync.RWMutex
)

func init() {
	go maintainNetworkSpeed()
}

func maintainNetworkSpeed() {
	for {
		downloadSpeed, uploadSpeed, err := getCurrentNetworkSpeed()
		if err == nil {
			speedMutex.Lock()
			currentSpeed = NetworkSpeed{
				Download: downloadSpeed,
				Upload:   uploadSpeed,
			}
			speedMutex.Unlock()
		}
		time.Sleep(time.Second)
	}
}

func getCurrentNetworkSpeed() (uint64, uint64, error) {
	initialStats, err := psnet.IOCounters(false)
	if err != nil {
		return 0, 0, err
	}

	time.Sleep(time.Second)

	finalStats, err := psnet.IOCounters(false)
	if err != nil {
		return 0, 0, err
	}

	downloadSpeed := finalStats[0].BytesRecv - initialStats[0].BytesRecv
	uploadSpeed := finalStats[0].BytesSent - initialStats[0].BytesSent

	return downloadSpeed, uploadSpeed, nil
}

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

func getServerStaticData() (ServerStaticData, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return ServerStaticData{}, err
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return ServerStaticData{}, err
	}

	diskInfo, err := getDiskInfo()
	if err != nil {
		return ServerStaticData{}, err
	}

	ipv4Supported, ipv6Supported := checkIPSupport()

	publicIpv4, publicIpv6, err := getPublicIP()
	if err != nil {
		publicIpv4 = "Unknown"
		publicIpv6 = "Unknown"
	}
	_, ipv4, _ := getLocalIPs()
	isNAT := checkNAT(publicIpv4, ipv4)

	data := ServerStaticData{
		ID:             NodeId,
		HostName:       hostInfo.Hostname,
		OSName:         hostInfo.OS,
		OSVersion:      hostInfo.PlatformVersion,
		Architecture:   hostInfo.KernelArch,
		Virtualization: hostInfo.VirtualizationSystem,
		PublicIPV4:     publicIpv4,
		PublicIPV6:     publicIpv6,
		NAT:            isNAT,
		VendorName:     "Unknown",
		CountryCode:    "Unknown",
		IPv4Supported:  ipv4Supported,
		IPv6Supported:  ipv6Supported,
		SwapTotal:      fmt.Sprint(memInfo.SwapTotal),
		MemoryTotal:    fmt.Sprint(memInfo.Total),
		DiskTotal:      fmt.Sprint(diskInfo.Total),
		UpDateTime:     time.Now().Format(time.RFC3339),
		Version:        version,
	}

	return data, nil
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

func ReportStatic() {
	staticData, err := getServerStaticData()
	if err != nil {
		fmt.Println("Error getting server static info", err)
		return
	}
	err = Report("api/report/static", staticData)
	if err != nil {
		fmt.Println("Error reporting static info", err)
	}
}

func ReportDynamic() {
	dynamicData, err := getServerDynamicData()
	if err != nil {
		fmt.Println("Error getting server dynamic info", err)
		return
	}
	// get actions
	actions := []string{}
	actionMap.Range(func(key, value any) bool {
		actions = append(actions, value.(*Action).String())
		return true
	})
	dynamicData.Actions = actions
	Report("api/report/dynamic", dynamicData)
}

func SafeReportDynamic() {
	for {
		ReportDynamic()
		time.Sleep(1 * time.Second)
	}
}

type ServerDynamicData struct {
	ID              string     `json:"id"`
	Load            [3]float64 `json:"load"`
	CPUUsage        float64    `json:"cpuUsage"`
	MemoryUsed      uint64     `json:"memoryUsed"`
	DiskUsed        uint64     `json:"diskUsed"`
	NetworkDownload uint64     `json:"networkDownload"`
	NetworkUpload   uint64     `json:"networkUpload"`
	TrafficDownload uint64     `json:"trafficDownload"`
	TrafficUpload   uint64     `json:"trafficUpload"`
	TCPCount        int        `json:"tcpCount"`
	UDPCount        int        `json:"udpCount"`
	ProcessCount    int        `json:"processCount"`
	ThreadCount     int        `json:"threadCount"`
	Actions         []string   `json:"actions"`
}

func getServerDynamicData() (ServerDynamicData, error) {
	loadAvg, err := load.Avg()
	if err != nil {
		return ServerDynamicData{}, err
	}

	cpuUsage, err := getCPUUsage()
	if err != nil {
		return ServerDynamicData{}, err
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return ServerDynamicData{}, err
	}

	diskInfo, err := getDiskInfo()
	if err != nil {
		return ServerDynamicData{}, err
	}

	networkInfo, err := getNetworkInfo()
	if err != nil {
		return ServerDynamicData{}, err
	}

	tcpCount, udpCount, err := getConnectionCounts()
	if err != nil {
		return ServerDynamicData{}, err
	}

	processCount, threadCount, err := getProcessAndThreadCounts()
	if err != nil {
		return ServerDynamicData{}, err
	}

	speedMutex.RLock()
	speed := currentSpeed
	speedMutex.RUnlock()

	data := ServerDynamicData{
		ID:              NodeId,
		Load:            [3]float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15},
		CPUUsage:        cpuUsage,
		MemoryUsed:      memInfo.Used,
		DiskUsed:        diskInfo.Used,
		NetworkDownload: speed.Download,
		NetworkUpload:   speed.Upload,
		TrafficUpload:   networkInfo.BytesSent,
		TrafficDownload: networkInfo.BytesRecv,
		TCPCount:        tcpCount,
		UDPCount:        udpCount,
		ProcessCount:    processCount,
		ThreadCount:     threadCount,
	}

	return data, nil
}

func ReportDetail() {

}

var (
	reportInterval = 1 * time.Second
	Host           = "http://127.0.0.1:8080" // 替换为实际的报告地址
	NodeId         = "default_test"
)

func ApiPath(path string) string {
	ret, _ := url.JoinPath(Host, path)
	return ret
}

type Action struct {
	Action string `json:"action"`
	Data   string `json:"data"`
	Name   string `json:"name"`
	Status string `json:"status"`
	PID    int    `json:"pid"`
	Time   string `json:"time"`
	Result string `json:"result"`
	Error  string `json:"error"`
}

func (a *Action) String() string {
	return fmt.Sprintf("Action{Action: %s, Data: %s, Name: %s, Status: %s, PID: %d, Time: %s, Result: %s, Error: %s}", a.Action, a.Data, a.Name, a.Status, a.PID, a.Time, a.Result, a.Error)
}

func (a *Action) IsRunning() bool {
	return a.Status == "running"
}

func (a *Action) TakeAction() {
	go SafeRun(func() {
		fmt.Println("taking action", a)
		a.Status = "running"
		a.Time = time.Now().Format(time.RFC3339)
		//run shell command of a.Data
		cmd := exec.Command("sh", "-c", a.Data)
		output, err := cmd.CombinedOutput()
		if err != nil {
			a.Result = "failed"
			a.Error = err.Error()
		} else {
			a.Result = string(output)
		}
		a.Status = "finished"
	})
}

var actionMap = sync.Map{}

func TakeAction(action *Action) {
	old, _ := actionMap.LoadOrStore(action.Action, action)
	if old != nil {
		fmt.Println("action already taken", action)
	}
	oldAction := old.(*Action)
	if oldAction.IsRunning() {
		return
	}
	oldAction.TakeAction()
}

func Report(path string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return err
	}
	fmt.Println("report", path, "with", string(jsonData))
	url := ApiPath(path)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	//parse action
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	action := &Action{}
	err = json.Unmarshal(body, &action)
	if err != nil {
		return err
	}
	TakeAction(action)
	fmt.Println("action", action)
	return nil
}

func SafeRun(call func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("Error sending report " + fmt.Sprintln(r))
		}
	}()
	call()
	return
}

func SafeReportStatic() {
	for {
		err := SafeRun(func() {
			ReportStatic()
		})
		if err != nil {
			fmt.Println("Reporting static data with err", err)
		}
		time.Sleep(1 * time.Second)
	}
}

func main() {
	flag.DurationVar(&reportInterval, "i", 1*time.Second, "Report interval")
	flag.Parse()

	args := flag.Args()
	if len(args) >= 1 {
		Host = args[0]
	}
	if len(args) >= 2 {
		NodeId = args[1]
	}

	go SafeReportStatic()
	go SafeReportDynamic()
	select {}
}

func getCPUUsage() (float64, error) {
	percentage, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, err
	}
	if len(percentage) > 0 {
		return percentage[0], nil
	}
	return 0, fmt.Errorf("no CPU usage data available")
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
		// fmt.Println(partition.Mountpoint, partition.String(), partition.Device)
		totalUsed += usage.Used
		totalSpace += usage.Total
	}

	return struct{ Used, Total uint64 }{totalUsed, totalSpace}, nil
}

func getNetworkInfo() (psnet.IOCountersStat, error) {
	ioCounters, err := psnet.IOCounters(false)
	if err != nil {
		return psnet.IOCountersStat{}, err
	}

	if len(ioCounters) > 0 {
		return ioCounters[0], nil
	}

	return psnet.IOCountersStat{}, fmt.Errorf("no network data available")
}

const (
	TCP uint32 = 1
	UDP uint32 = 2
)

func getConnectionCounts() (int, int, error) {
	conns, err := psnet.Connections("all")
	if err != nil {
		return 0, 0, err
	}

	tcpCount := 0
	udpCount := 0
	for _, conn := range conns {
		switch conn.Type {
		case TCP:
			tcpCount++
		case UDP:
			udpCount++
		}
	}

	return tcpCount, udpCount, nil
}

func getProcessAndThreadCounts() (int, int, error) {
	processes, err := process.Processes()
	if err != nil {
		return 0, 0, err
	}

	processCount := len(processes)
	threadCount := 0

	for _, p := range processes {
		numThreads, err := p.NumThreads()
		if err == nil {
			threadCount += int(numThreads)
		}
	}

	return processCount, threadCount, nil
}

func checkIPSupport() (bool, bool) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false, false
	}

	ipv4Supported := false
	ipv6Supported := false

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ipnet.IP.To4() != nil {
				ipv4Supported = true
			} else if ipnet.IP.To16() != nil {
				ipv6Supported = true
			}
		}
	}

	return ipv4Supported, ipv6Supported
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
}

func getIP(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(ip), nil
}

func getPublicIP() (string, string, error) {
	ipv4, err := getIP("https://api.ipify.org")
	if err != nil {
		ipv4 = "Unknown"
	}

	ipv6, err := getIP("https://api6.ipify.org")
	if err != nil {
		ipv6 = "Unknown"
	}

	return ipv4, ipv6, nil
}
