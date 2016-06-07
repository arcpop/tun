// +build linux

package tun

import (
    "syscall"
	"os"
    "net"
    "strings"
    "errors"
    "unsafe"
)
const (
	IFF_TUN   = 0x0001
	IFF_TAP   = 0x0002
	IFF_NO_PI = 0x1000
)

var (
    _ TunInterface = &tunInterface{}
)

type tunInterface struct {
    file *os.File
    name string
}

type ifreq_flags struct {
    ifnam [16]byte
    flags uint16
}

type ifreq_addr struct {
    ifnam [16]byte
    addr syscall.SockaddrInet4
}

type ifreq_mtu struct {
    ifnam [16]byte
    mtu int32
}

func newTun(ifaceName string) (TunInterface, error) {
    var req ifreq_flags
    file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

    copy(req.ifnam[:], ifaceName)
    req.ifnam[15] = 0
    req.flags = IFF_TUN | IFF_NO_PI
    err = ioctl(file, syscall.TUNSETIFF, uintptr(unsafe.Pointer(&req)))
	if err != nil {
        file.Close()
		return nil, err
	}
    ifaceName = strings.Trim(string(req.ifnam[:]), "\x00")
    req.flags = 0
    err = ioctl(file, syscall.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&req)))
    if err != nil {
        file.Close()
		return nil, err
	}
    req.flags |= syscall.IFF_UP
    err = ioctl(file, syscall.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&req)))
    if err != nil {
        file.Close()
		return nil, err
	}

	iface := &tunInterface{ 
        file: file, 
        name: ifaceName,
    }

	return iface, nil
}

func (t *tunInterface) SetIPAddress(ip, broadcast net.IP, netmask net.IP) error {
    var req ifreq_addr
    var req2 ifreq_flags
    ipv4 := ip.To4()
    broadcast4 := broadcast.To4()
    netmask4 := netmask.To4()
    if ipv4 == nil || 
        ((broadcast != nil) && (broadcast4 == nil)) || 
        netmask4 == nil {
        return errors.New("IPv6 not yet implemented")
    }
    copy(req.ifnam[:], t.name)
    req.ifnam[15] = 0
    copy(req.addr.Addr[:], ipv4[:])
    req.addr.Port = 0
    err := ioctl(t.file, syscall.SIOCSIFADDR, uintptr(unsafe.Pointer(&req)))
    if err != nil {
		return err
	}

    copy(req.addr.Addr[:], ipv4[:])
    err = ioctl(t.file, syscall.SIOCSIFADDR, uintptr(unsafe.Pointer(&req)))
    if err != nil {
		return err
	}

    if broadcast4 == nil {
        return 
    }

    //First set the broadcast address
    copy(req.addr.Addr[:], broadcast4[:])
    err = ioctl(t.file, syscall.SIOCSIFBRDADDR, uintptr(unsafe.Pointer(&req)))
    if err != nil {
		return err
	}

    //Then indicate with flags that a valid broadcast address is set
    copy(req2.ifnam[:], t.name)
    req2.ifnam[15] = 0
    err = ioctl(file, syscall.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&req2)))
    if err != nil {
		return err
	}
    req2.flags |= syscall.IFF_BROADCAST
    err = ioctl(file, syscall.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&req2)))
    if err != nil {
		return err
	}

    return nil
}

func (t *tunInterface) SetMTU(mtu int) error {
    var req ifreq_mtu
    copy(req.ifnam[:], t.name)
    req.ifnam[15] = 0
    req.mtu = mtu
    err := ioctl(file, syscall.SIOCSIFMTU, uintptr(unsafe.Pointer(&req)))
    if err != nil {
		return err
	}
}

func (t *tunInterface) Read(p []byte) (n int, err error) {
    return t.file.Read(p)
}

func (t *tunInterface) Write(p []byte) (n int, err error) {
    return t.file.Write(p)
}

func (t *tunInterface) Close() error {
    return t.file.Close()
}

func (t* tunInterface) GetName() string {
    return t.name
}

func ioctl(file *os.File, cmd int, arg uintptr) error {
    _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(cmd), arg)
    if errno != 0 {
        return errno
    }
    return nil
}

