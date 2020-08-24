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

	fwd := r.Header.Get("X-Forwarded-For")
	if fwd != "" {
		info.IP = strings.Split(fwd, ",")[0]
		if strings.Index(info.IP, ":") == -1 {
			info.Type = "ipv4"
		} else {
			info.Type = "ipv6"
		}
	} else {
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
	}

	json.NewEncoder(w).Encode(info)
}
