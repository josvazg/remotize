package dep

// Sample of a simple Name Service
type NS struct {
	name2ip map[string]string
	ip2name map[string]string
}

func (ns *NS) IpFor(hostname string) string {
	return ns.name2ip[hostname]
}

func (ns *NS) NameFor(ip string) string {
	return ns.ip2name[ip]
}

func (ns *NS) Bind(hostname, ip string) {
	ns.name2ip[hostname] = ip
	ns.ip2name[ip] = hostname
}

// Sample for a simple content driven Blob Server
type BlobServer interface {
	Put(hashid string, content []byte)
	Get(hashid string) []byte
	List() []string
}
