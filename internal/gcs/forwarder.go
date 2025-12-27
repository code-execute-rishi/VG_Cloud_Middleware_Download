package gcs

import (
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

type Endpoint struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Port      int    `json:"port"`
	Enabled   bool   `json:"enabled"`
	Video     bool   `json:"enable_video"`
	Telemetry bool   `json:"enable_telemetry"`
}

type Forwarder struct {
	mu           sync.RWMutex
	endpoints    map[string]*Endpoint
	conn         *net.UDPConn
	mavlinkAddr  *net.UDPAddr
	proxyAddress string
}

func fmtAddress(ip string, port int) string {
	return strings.Join([]string{ip, strconv.Itoa(port)}, ":")
}

func NewForwarder(listenAddr string) *Forwarder {
	addr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		log.Printf("[GCS] Invalid proxy address: %v", err)
		return nil
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Printf("[GCS] Failed to bind proxy: %v", err)
		return nil
	}

	f := &Forwarder{
		endpoints:    make(map[string]*Endpoint),
		conn:         conn,
		proxyAddress: listenAddr,
	}

	go f.run()
	log.Printf("[GCS] Proxy listening on %s", listenAddr)
	return f
}

func (f *Forwarder) run() {
	buf := make([]byte, 4096)
	for {
		n, src, err := f.conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("[GCS] Read error: %v", err)
			continue
		}

		// Heuristic: If src is local and NOT from our known GCS endpoints, assume it is gomavlib
		isLocal := src.IP.IsLoopback()

		// If we haven't identified mavlink source yet, or it's definitely local and not a GCS target
		if isLocal && (f.mavlinkAddr == nil || f.mavlinkAddr.String() == src.String()) {
			f.mavlinkAddr = src
			f.broadcast(buf[:n])
		} else {
			// Assume it's from a Remote GCS, sending back to Mavlink Node
			if f.mavlinkAddr != nil {
				f.conn.WriteToUDP(buf[:n], f.mavlinkAddr)
			}
		}
	}
}

func (f *Forwarder) broadcast(data []byte) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for _, ep := range f.endpoints {
		if ep.Enabled && ep.Telemetry {
			addr, err := net.ResolveUDPAddr("udp", fmtAddress(ep.IP, ep.Port))
			if err == nil {
				f.conn.WriteToUDP(data, addr)
			}
		}
	}
}

func (f *Forwarder) AddEndpoint(ep Endpoint) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.endpoints[ep.ID] = &ep
	log.Printf("[GCS] Added Endpoint: %s (%s:%d)", ep.Name, ep.IP, ep.Port)
}

func (f *Forwarder) RemoveEndpoint(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.endpoints, id)
	log.Printf("[GCS] Removed Endpoint: %s", id)
}

func (f *Forwarder) ListEndpoints() []Endpoint {
	f.mu.RLock()
	defer f.mu.RUnlock()
	list := []Endpoint{}
	for _, ep := range f.endpoints {
		list = append(list, *ep)
	}
	return list
}

func (f *Forwarder) ToggleEndpoint(id string, enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if ep, ok := f.endpoints[id]; ok {
		ep.Enabled = enabled
		log.Printf("[GCS] Endpoint %s status: %v", ep.Name, enabled)
	}
}

// SyncEndpoints replaces the current list with the new list, preserving state where possible
func (f *Forwarder) SyncEndpoints(newEndpoints []Endpoint) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track which IDs we've seen in the new list
	seen := make(map[string]bool)

	for _, newEp := range newEndpoints {
		seen[newEp.ID] = true

		// check if exists
		if existing, ok := f.endpoints[newEp.ID]; ok {
			// Update fields if changed (simple equality check)
			if existing.IP != newEp.IP || existing.Port != newEp.Port || existing.Enabled != newEp.Enabled || existing.Name != newEp.Name {
				log.Printf("[GCS] Updating Endpoint %s: %s -> %s", newEp.ID, existing.Name, newEp.Name)
				f.endpoints[newEp.ID] = &newEp
			}
		} else {
			// Add new
			log.Printf("[GCS] Synced New Endpoint: %s (%s:%d)", newEp.Name, newEp.IP, newEp.Port)
			f.endpoints[newEp.ID] = &newEp
		}
	}

	// Remove any that are not in the new list
	for id := range f.endpoints {
		if !seen[id] {
			log.Printf("[GCS] Removing Stale Endpoint: %s", id)
			delete(f.endpoints, id)
		}
	}
}
