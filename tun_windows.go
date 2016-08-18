// +build windows
package tun

import (
    "golang.org/x/sys/windows"
    "golang.org/x/sys/windows/registry"
	"errors"
	"net"
)

type tunWindows struct {
    guid string
    file windows.Handle
}

func newTun() (TunInterface, error) {
    guid, err := getAdapterGUID()
    if err != nil {
        return nil, err
    }

    fd, err := windows.CreateFile(
        windows.StringToUTF16Ptr("\\\\.\\Global\\" + guid + ".tap"), 
        windows.GENERIC_READ | windows.GENERIC_WRITE,
        windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE,
        nil,
        windows.OPEN_EXISTING, 
        windows.FILE_ATTRIBUTE_SYSTEM,
        0)
    if err != nil {
        return nil, err
    }

    return &tunWindows{
        guid: guid,
        file: fd,
    }, nil
}

func (t *tunWindows) Close() error {
    return windows.Close(t.file)
}

func (t *tunWindows) GetName() string {
    
}

func (t *tunWindows) Read(b []byte) (int, error) {
    return windows.Read(t.file, b)
}

func (t *tunWindows) SetIPAddress(ip, broadcast net.IP, netmask net.IP) error {

}

func (t *tunWindows) SetMTU(mtu int) error {
    //HKLM\System\CurrentControlSet\Services\Tcpip\Parameters\Interfaces\<guid> set MTU ->
}

func (t *tunWindows) Write(b []byte) (int, error) {
    return windows.Write(t.file, b)
}


func getAdapterGUID() (guid string, err error) {
    var rootKey, adapterKey registry.Key
    rootKey, err = registry.OpenKey(
        registry.LOCAL_MACHINE, 
        "SYSTEM\\CurrentControlSet\\Control\\Class\\{4D36E972-E325-11CE-BFC1-08002BE10318}", 
        registry.READ | registry.ENUMERATE_SUB_KEYS)
    if err != nil {
        return
    }
    defer rootKey.Close()

    var subKeyNames []string
    subKeyNames, err = rootKey.ReadSubKeyNames(0)
    if err != nil {
        return
    }

    for _, n := range subKeyNames {
        var subKey registry.Key
        subKey, err = registry.OpenKey(rootKey, n, registry.QUERY_VALUE | registry.READ)
        if err != nil {
            continue
        }
        var v string
        v, _, err = subKey.GetStringValue("ComponentId")
        if err != nil {
            subKey.Close()
            continue
        }
        if v != "tap0901" {
            subKey.Close()
            continue
        }
        v, _, err = subKey.GetStringValue("ComponentId")
        if err != nil {
            subKey.Close()
            continue
        }
        subKey.Close()
        guid = v
        return guid, nil
    }
    return "", errors.New("TAP adapter not found in registry")
}