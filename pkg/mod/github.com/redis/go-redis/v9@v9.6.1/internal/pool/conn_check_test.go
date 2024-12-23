//go:build linux || darwin || dragonfly || freebsd || netbsd || openbsd || solaris || illumos

package pool

import (
	"crypto/tls"
	"net"
	"net/http/httptest"
	"syscall"
	"time"

	. "github.com/bsm/ginkgo/v2"
	. "github.com/bsm/gomega"
)

var _ = Describe("tests conn_check with real conns", func() {
	var ts *httptest.Server
	var conn net.Conn
	var tlsConn *tls.Conn
	var sysConn syscall.Conn
	var tlsSysConn syscall.Conn
	var err error

	BeforeEach(func() {
		ts = httptest.NewServer(nil)
		conn, err = net.DialTimeout(ts.Listener.Addr().Network(), ts.Listener.Addr().String(), time.Second)
		Expect(err).NotTo(HaveOccurred())
		sysConn = conn.(syscall.Conn)
		tlsTestServer := httptest.NewUnstartedServer(nil)
		tlsTestServer.StartTLS()
		tlsConn, err = tls.DialWithDialer(&net.Dialer{Timeout: time.Second}, tlsTestServer.Listener.Addr().Network(), tlsTestServer.Listener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
		Expect(err).NotTo(HaveOccurred())
		tlsSysConn = tlsConn.NetConn().(syscall.Conn)
	})

	AfterEach(func() {
		ts.Close()
	})

	It("good conn check", func() {
		Expect(connCheck(sysConn)).NotTo(HaveOccurred())

		Expect(conn.Close()).NotTo(HaveOccurred())
		Expect(connCheck(sysConn)).To(HaveOccurred())
	})

	It("good tls conn check", func() {
		Expect(connCheck(tlsSysConn)).NotTo(HaveOccurred())

		Expect(tlsConn.Close()).NotTo(HaveOccurred())
		Expect(connCheck(tlsSysConn)).To(HaveOccurred())
	})

	It("bad conn check", func() {
		Expect(conn.Close()).NotTo(HaveOccurred())
		Expect(connCheck(sysConn)).To(HaveOccurred())
	})

	It("bad tls conn check", func() {
		Expect(tlsConn.Close()).NotTo(HaveOccurred())
		Expect(connCheck(tlsSysConn)).To(HaveOccurred())
	})

	It("check conn deadline", func() {
		Expect(conn.SetDeadline(time.Now())).NotTo(HaveOccurred())
		time.Sleep(time.Millisecond * 10)
		Expect(connCheck(sysConn)).To(HaveOccurred())

		Expect(conn.SetDeadline(time.Now().Add(time.Minute))).NotTo(HaveOccurred())
		time.Sleep(time.Millisecond * 10)
		Expect(connCheck(sysConn)).NotTo(HaveOccurred())
		Expect(conn.Close()).NotTo(HaveOccurred())
	})
})
