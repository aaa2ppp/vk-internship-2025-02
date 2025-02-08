package main

import "time"

type Host struct {
	ID   int    `json:"host_id"`
	Name string `json:"host_name"`
}

type PingResult struct {
	HostID   int           `json:"host_id,omitempty"`
	HostName string        `json:"host_name"`
	IP       string        `json:"ip"`
	Time     time.Time     `json:"time"`
	Rtt      time.Duration `json:"rtt"`
	Success  bool          `json:"success"`
}
