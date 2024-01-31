package main

import (
	"time"
	"net"
	"strings"
)

func StrTrim(str string) string {
	return strings.Trim(str, " \n\r")
}
func GetDateTime(withTime bool, timestamp int64) string {
	formatStr := "2006-01-02"
	if withTime {
		formatStr += " 15:04:05"
	}
	var curTime time.Time
	if timestamp > 0 {
		curTime = time.Unix(timestamp, 0)
	} else {
		curTime = time.Now()
	}
	return curTime.Format(formatStr)
}
func CheckPrivateIP(ip string) bool {
	ipParsed := net.ParseIP(ip)
	return ipParsed.IsPrivate()
}
