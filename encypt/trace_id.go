package encypt

func CrateTraceId(ip string) (traceId string) {
	var now = time.Now()
	var timestamp = uint32(now.Unix())
	var timeNano = now.UnixNano()
	var pid = os.Getpid()
	var b = bytes.Buffer{}
	var netIP = net.ParseIP(ip)

	if netIP == nil {
		b.WriteString("00000000")
	} else {
		b.WriteString(hex.EncodeToString(netIP.To4()))
	}
	b.WriteString(fmt.Sprintf("%08x", timestamp&0xffffffff))
	b.WriteString(fmt.Sprintf("%04x", timeNano&0xffff))
	b.WriteString(fmt.Sprintf("%04x", pid&0xffff))
	b.WriteString(fmt.Sprintf("%06x", rand.Int31n(1<<24)))
	b.WriteString("b0") // 末两位标记来源,b0为go

	return b.String()
}
