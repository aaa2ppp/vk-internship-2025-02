package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"time"
)

type PingResult struct {
	IP       string    `json:"ip"`
	PingTime time.Time `json:"ping_time"`
	Success  bool      `json:"success"`
}

func pingIP(ip string) bool {
	cmd := exec.Command("ping", "-c", "1", "-W", "1", ip)
	err := cmd.Run()
	return err == nil
}

func main() {
	for {
		ips := getDockerContainerIPs()
		for _, ip := range ips {
			success := pingIP(ip)
			result := PingResult{
				IP:       ip,
				PingTime: time.Now(),
				Success:  success,
			}
			sendPingResult(result)
		}
		time.Sleep(10 * time.Second)
	}
}

func getDockerContainerIPs() []string {
	// Implement logic to get Docker container IPs
	return []string{"172.17.0.2", "172.17.0.3"}
}

func sendPingResult(result PingResult) {
	jsonData, err := json.Marshal(result)
	if err != nil {
		log.Fatal(err)
	}

	_, err = http.Post("http://backend:8080/add-ping-result", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal(err)
	}
}
