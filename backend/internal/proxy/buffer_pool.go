package proxy

import "sync"

// Buffer pools for reuse across proxies
var (
	// tcpBufferPool reuses 32KB buffers for TCP proxying
	tcpBufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 32768) // 32KB default
			return &buf
		},
	}

	// udpBufferPool reuses buffers for UDP packets
	udpBufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 65507) // Max UDP packet size
			return &buf
		},
	}
)

// getTCPBuffer retrieves a buffer from the TCP pool
func getTCPBuffer() *[]byte {
	return tcpBufferPool.Get().(*[]byte)
}

// putTCPBuffer returns a buffer to the TCP pool
func putTCPBuffer(buf *[]byte) {
	tcpBufferPool.Put(buf)
}

// getUDPBuffer retrieves a buffer from the UDP pool
func getUDPBuffer() *[]byte {
	return udpBufferPool.Get().(*[]byte)
}

// putUDPBuffer returns a buffer to the UDP pool
func putUDPBuffer(buf *[]byte) {
	udpBufferPool.Put(buf)
}
