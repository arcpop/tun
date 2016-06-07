package tun

import (
	"net"
    "io"
)


type TunInterface interface {
    io.ReadWriteCloser
    SetIPAddress(ip, broadcast net.IP, netmask net.IP) error 
    SetMTU(mtu int) error 
    GetName() string
}
func New(name string) (TunInterface, error) {
    return newTun(name)
}