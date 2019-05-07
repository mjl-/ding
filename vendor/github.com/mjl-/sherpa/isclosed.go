// +build !plan9

package sherpa

import (
	"net"
	"syscall"
)

func isConnectionClosed(err error) bool {
	oe, ok := err.(*net.OpError)
	return ok && (oe.Err == syscall.EPIPE || oe.Err == syscall.ECONNRESET)
}
