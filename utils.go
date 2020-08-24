package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Info struct {
	IP   string `json:"ip"`
	Type string `json:"type"`
}

func info(w http.ResponseWriter, r *http.Request) {
	info := Info{}

	spl := strings.Split(r.RemoteAddr, "]:")
	if len(spl) > 1 {
		info.IP = spl[0][1:]
		info.Type = "ipv6"
	} else {
		spl = strings.Split(r.RemoteAddr, ":")
		if len(spl) > 1 {
			info.IP = spl[0]
			info.Type = "ipv4"
		}
	}

	json.NewEncoder(w).Encode(info)
}
