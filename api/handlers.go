package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-ping/ping"
)

func PingHandler(c *gin.Context) {
	var req struct {
		Host      string `json:"host"`
		Count     int    `json:"count"`
		TimeoutMs int    `json:"timeout_ms"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid json"})
		return
	}
	if req.Count <= 0 {
		req.Count = 4
	}
	if req.TimeoutMs <= 0 {
		req.TimeoutMs = 2000
	}
	res, err := runICMPPing(req.Host, req.Count, time.Duration(req.TimeoutMs)*time.Millisecond)
	if err != nil {
		tres, terr := runTCPing(req.Host, 80, time.Duration(req.TimeoutMs)*time.Millisecond, req.Count)
		if terr != nil {
			c.JSON(500, gin.H{"error": "ping failed", "detail": err.Error() + " | " + terr.Error()})
			return
		}
		c.JSON(200, gin.H{"method": "tcp-fallback", "result": tres})
		return
	}
	c.JSON(200, gin.H{"method": "icmp", "result": res})
}

func runICMPPing(host string, count int, timeout time.Duration) (map[string]any, error) {
	pinger, err := ping.NewPinger(host)
	if err != nil {
		return nil, err
	}
	pinger.Count = count
	pinger.Timeout = time.Duration(count)*timeout + 2*time.Second
	pinger.SetPrivileged(true)
	err = pinger.Run()
	if err != nil {
		return nil, err
	}
	stats := pinger.Statistics()
	return map[string]any{
		"packets_sent":     stats.PacketsSent,
		"packets_recv":     stats.PacketsRecv,
		"packet_loss":      stats.PacketLoss,
		"avg_rtt_ms":       stats.AvgRtt.Milliseconds(),
		"min_rtt_ms":       stats.MinRtt.Milliseconds(),
		"max_rtt_ms":       stats.MaxRtt.Milliseconds(),
		"stddev_ms":        stats.StdDevRtt.Milliseconds(),
		"duration_seconds": stats.Duration.Seconds(),
	}, nil
}

func runTCPing(host string, port int, timeout time.Duration, count int) (map[string]any, error) {
	var success int
	var sumRTT time.Duration
	var min, max time.Duration
	min = time.Hour
	for i := 0; i < count; i++ {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, itoa(port)), timeout)
		rtt := time.Since(start)
		if err == nil {
			success++
			sumRTT += rtt
			if rtt < min {
				min = rtt
			}
			if rtt > max {
				max = rtt
			}
			conn.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	if success == 0 {
		return map[string]any{
			"packets_sent": count,
			"packets_recv": success,
			"packet_loss":  100,
		}, errors.New("no tcp success")
	}
	avg := sumRTT / time.Duration(success)
	return map[string]any{
		"packets_sent": count,
		"packets_recv": success,
		"packet_loss":  float64(count-success) / float64(count) * 100,
		"avg_rtt_ms":   avg.Milliseconds(),
		"min_rtt_ms":   min.Milliseconds(),
		"max_rtt_ms":   max.Milliseconds(),
	}, nil
}

func TCPingHandler(c *gin.Context) {
	var req struct {
		Host      string `json:"host"`
		Port      int    `json:"port"`
		TimeoutMs int    `json:"timeout_ms"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid json"})
		return
	}
	if req.Port == 0 {
		req.Port = 443
	}
	if req.TimeoutMs <= 0 {
		req.TimeoutMs = 2000
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.TimeoutMs)*time.Millisecond)
	defer cancel()
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(req.Host, itoa(req.Port)))
	if err != nil {
		c.JSON(200, gin.H{"ok": false, "error": err.Error()})
		return
	}
	conn.Close()
	c.JSON(200, gin.H{"ok": true})
}

func TracerouteHandler(c *gin.Context) {
	var req struct {
		Host      string `json:"host"`
		MaxHops   int    `json:"max_hops"`
		TimeoutMs int    `json:"timeout_ms"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid json"})
		return
	}
	if req.MaxHops <= 0 {
		req.MaxHops = 30
	}
	if req.TimeoutMs <= 0 {
		req.TimeoutMs = 3000
	}
	out, err := runTracerouteCmd(req.Host, req.MaxHops, req.TimeoutMs)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"output": out})
}

func runTracerouteCmd(host string, maxhops int, timeoutMs int) (string, error) {
	cmd := []string{"traceroute", "-m", itoa(maxhops), "-w", itoa(timeoutMs/1000), host}
	outb, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	return string(outb), err
}

func DNSHandler(c *gin.Context) {
	var req struct {
		Host string `json:"host"`
		Type string `json:"type"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid json"})
		return
	}
	t := strings.ToUpper(req.Type)
	switch t {
	case "A", "":
		ips, err := net.LookupHost(req.Host)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"A": ips})
	case "MX":
		mx, err := net.LookupMX(req.Host)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"MX": mx})
	case "NS":
		ns, err := net.LookupNS(req.Host)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"NS": ns})
	case "TXT":
		txt, err := net.LookupTXT(req.Host)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"TXT": txt})
	default:
		c.JSON(400, gin.H{"error": "unsupported type"})
	}
}

func SpeedHandler(c *gin.Context) {
	var req struct {
		URL       string `json:"url"`
		DurationS int    `json:"duration_s"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid json"})
		return
	}
	if req.DurationS <= 0 {
		req.DurationS = 5
	}
	result, err := runHTTPDownloadTest(req.URL, time.Duration(req.DurationS)*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, result)
}

func runHTTPDownloadTest(url string, duration time.Duration) (map[string]any, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var total int64
	buf := make([]byte, 32*1024)
	start := time.Now()
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			total += int64(n)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			if ctx.Err() != nil {
				break
			}
			return nil, err
		}
	}
	elapsed := time.Since(start).Seconds()
	return map[string]any{
		"status":         resp.StatusCode,
		"bytes_download": total,
		"seconds":        elapsed,
		"throughput_bps": float64(total) * 8.0 / elapsed,
	}, nil
}

func BulkPingHandler(c *gin.Context) {
	var req struct {
		Hosts     []string `json:"hosts"`
		Count     int      `json:"count"`
		TimeoutMs int      `json:"timeout_ms"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid json"})
		return
	}
	if len(req.Hosts) == 0 {
		c.JSON(400, gin.H{"error": "no hosts"})
		return
	}
	if req.Count <= 0 {
		req.Count = 3
	}
	if req.TimeoutMs <= 0 {
		req.TimeoutMs = 2000
	}
	type entry struct {
		Host string                 `json:"host"`
		Out  map[string]interface{} `json:"out"`
		Err  string                 `json:"err,omitempty"`
	}
	ch := make(chan entry, len(req.Hosts))
	sem := make(chan struct{}, 25)
	for _, h := range req.Hosts {
		h := h
		go func() {
			sem <- struct{}{}
			out, err := runICMPPing(h, req.Count, time.Duration(req.TimeoutMs)*time.Millisecond)
			var e entry
			e.Host = h
			if err != nil {
				tout, terr := runTCPing(h, 80, time.Duration(req.TimeoutMs)*time.Millisecond, req.Count)
				if terr != nil {
					e.Err = err.Error() + " | " + terr.Error()
				} else {
					e.Out = tout
				}
			} else {
				e.Out = out
			}
			ch <- e
			<-sem
		}()
	}
	var resp []entry
	for i := 0; i < len(req.Hosts); i++ {
		resp = append(resp, <-ch)
	}
	c.JSON(200, gin.H{"results": resp})
}

func BulkHTMLHandler(c *gin.Context) {
	var req struct {
		URLs      []string `json:"urls"`
		Keyword   string   `json:"keyword"`
		TimeoutMs int      `json:"timeout_ms"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid json"})
		return
	}
	if req.TimeoutMs <= 0 {
		req.TimeoutMs = 5000
	}
	client := &http.Client{Timeout: time.Duration(req.TimeoutMs) * time.Millisecond}
	type res struct {
		URL        string `json:"url"`
		StatusCode int    `json:"status_code"`
		HasKeyword bool   `json:"has_keyword"`
		DurationMs int64  `json:"duration_ms"`
		Error      string `json:"error,omitempty"`
	}
	ch := make(chan res, len(req.URLs))
	sem := make(chan struct{}, 20)
	for _, u := range req.URLs {
		u := u
		go func() {
			sem <- struct{}{}
			start := time.Now()
			r := res{URL: u}
			resp, err := client.Get(u)
			if err != nil {
				r.Error = err.Error()
				ch <- r
				<-sem
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*200))
			r.StatusCode = resp.StatusCode
			if req.Keyword != "" && strings.Contains(string(body), req.Keyword) {
				r.HasKeyword = true
			}
			r.DurationMs = time.Since(start).Milliseconds()
			ch <- r
			<-sem
		}()
	}
	var out []res
	for i := 0; i < len(req.URLs); i++ {
		out = append(out, <-ch)
	}
	c.JSON(200, gin.H{"results": out})
}

func itoa(i int) string { return strconv.Itoa(i) }