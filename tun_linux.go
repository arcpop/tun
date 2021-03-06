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
    index int
    mtu int
    sockfd uintptr
}

type ifreq_flags struct {
    ifnam [16]byte
    flags uint16
}
type sockaddr_in struct {
    sin_family int16
    sin_port int16
    sin_addr [4]byte
    sin_zero [8]byte
}
type ifreq_addr struct {
    ifnam [16]byte
    addr sockaddr_in
}

type ifreq_index struct {
    ifnam [16]byte
    index int32
}

type ifreq_mtu struct {
    ifnam [16]byte
    mtu int32
}

func newTun(ifaceName string) (TunInterface, error) {
    var req ifreq_flags
    file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
        println("Failed to open file")
		return nil, err
	}

    copy(req.ifnam[:], ifaceName)
    req.ifnam[15] = 0
    req.flags = IFF_TUN | IFF_NO_PI
    err = ioctl(file.Fd(), syscall.TUNSETIFF, uintptr(unsafe.Pointer(&req)))
	if err != nil {
        println("ioctl 1 failed")
        file.Close()
		return nil, err
	}
    ifaceName = strings.Trim(string(req.ifnam[:]), "\x00")
    println("Interface: " + ifaceName)
    netif, err := net.InterfaceByName(ifaceName)
    if err != nil {
        file.Close()
		return nil, err
    }

    sockfd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
    if err != nil {
        file.Close()
		return nil, err
    }

	iface := &tunInterface{ 
        file: file, 
        name: ifaceName,
        index: netif.Index,
        mtu: netif.MTU,
        sockfd: uintptr(sockfd),
    }

	return iface, nil
}

func (t *tunInterface) setFlags(flags uint16) error {
    var req ifreq_flags
    
    copy(req.ifnam[:], t.name)
    req.ifnam[15] = 0
    req.flags = 0
    err := ioctl(t.sockfd, syscall.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&req)))
    if err != nil {
		return err
	}
    req.flags |= flags
    err = ioctl(t.sockfd, syscall.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&req)))
    if err != nil {
		return err
	}
    return nil
}

func (t *tunInterface) SetIPAddress(ip, broadcast net.IP, netmask net.IP) error {
    var req ifreq_addr
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


    req.addr.sin_family = syscall.AF_INET
    copy(req.addr.sin_addr[:], ipv4[:])
    req.addr.sin_port = 0
    err := ioctl(t.sockfd, syscall.SIOCSIFADDR, uintptr(unsafe.Pointer(&req)))
    if err != nil {
		return err
	}

    req.addr.sin_family = syscall.AF_INET
    copy(req.addr.sin_addr[:], netmask4[:])
    req.addr.sin_port = 0
    err = ioctl(t.sockfd, syscall.SIOCSIFNETMASK, uintptr(unsafe.Pointer(&req)))
    if err != nil {
		return err
	}

    if broadcast4 == nil {
        return t.setFlags(syscall.IFF_UP | syscall.IFF_RUNNING)
    }

    //First set the broadcast address
    req.addr.sin_family = syscall.AF_INET
    copy(req.addr.sin_addr[:], broadcast4[:])
    req.addr.sin_port = 0
    err = ioctl(t.sockfd, syscall.SIOCSIFBRDADDR, uintptr(unsafe.Pointer(&req)))
    if err != nil {
		return err
	}
    return t.setFlags(syscall.IFF_UP | syscall.IFF_RUNNING | syscall.IFF_BROADCAST)
}

func (t *tunInterface) SetMTU(mtu int) error {
    var req ifreq_mtu
    copy(req.ifnam[:], t.name)
    req.ifnam[15] = 0
    req.mtu = int32(mtu)
    err := ioctl(t.sockfd, syscall.SIOCSIFMTU, uintptr(unsafe.Pointer(&req)))
    if err != nil {
		return err
	}
    return nil
}

func (t *tunInterface) Read(p []byte) (n int, err error) {
    return t.file.Read(p)
}

func (t *tunInterface) Write(p []byte) (n int, err error) {
    return t.file.Write(p)
}

func (t *tunInterface) Close() error {
    syscall.Close(int(t.sockfd))
    return t.file.Close()
}

func (t* tunInterface) GetName() string {
    return t.name
}

func ioctl(fd uintptr, cmd uint, arg uintptr) error {
    _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(cmd), arg)
    if errno != 0 {
        return errno
    }
    return nil
}

