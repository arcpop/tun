// +build windows
package tun

import (
    "golang.org/x/sys/windows"
    "golang.org/x/sys/windows/registry"
	"errors"
	"net"
	"os/exec"
	"strconv"
)

var (
    ctlTAP_WIN_IOCTL_SET_MEDIA_STATUS = ctlCode(0x22, 6, 0, 0)
    ctlTAP_WIN_IOCTL_CONFIG_TUN = ctlCode(0x22, 10, 0, 0)
    errAdapterNotConfigured = errors.New("Adapter used but not yet configured")
)

type tunWindows struct {
    guid, humanReadableName string
    file windows.Handle
    ready bool
}

func newTun() (TunInterface, error) {
    guid, err := getAdapterGUID()
    if err != nil {
        return nil, err
    }
    name, err := getHumanReadableName(guid)
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
        humanReadableName: name,
        guid: guid,
        file: fd,
        ready: false,
    }, nil
}

func (t *tunWindows) Close() error {
    t.ready = false
    return windows.Close(t.file)
}

func (t *tunWindows) GetName() string {
    return t.humanReadableName
}

func (t *tunWindows) Read(b []byte) (int, error) {
    if t.ready {
        return windows.Read(t.file, b)
    }
    return 0, errAdapterNotConfigured
}

func (t *tunWindows) SetIPAddress(ip, broadcast, netmask net.IP) error {
    var buf [12]byte
    var returned uint32
    
    ip4 := ip.To4()
    netmask4 := netmask.To4()
    
    if ip4 == nil || netmask4 == nil {
        return errors.New("Invalid network addresses")
    }
    
    copy(buf[0:4], ip4[0:4])
    buf[4] = ip4[0] & netmask4[0]
    buf[5] = ip4[1] & netmask4[1]
    buf[6] = ip4[2] & netmask4[2]
    buf[7] = ip4[3] & netmask4[3]
    copy(buf[8:12], netmask4[0:4])
    
    err := windows.DeviceIoControl(t.file, ctlTAP_WIN_IOCTL_CONFIG_TUN, &buf[0], 12, nil, 0, &returned, nil)
    if err != nil {
        return err
    }
    buf[0] = 1
    buf[1] = 0
    buf[2] = 0
    buf[3] = 0
    return windows.DeviceIoControl(t.file, ctlTAP_WIN_IOCTL_SET_MEDIA_STATUS, &buf[0], 4, nil, 0, &returned, nil)
}

func (t *tunWindows) SetMTU(mtu int) error {
    //HKLM\System\CurrentControlSet\Services\Tcpip\Parameters\Interfaces\<guid> set mtu
    /*k, err := registry.OpenKey(
        registry.LOCAL_MACHINE,
        "System\\CurrentControlSet\\Services\\Tcpip\\Parameters\\Interfaces\\" + t.guid,
        registry.SET_VALUE)
    if err != nil {
        return err
    }
    defer k.Close()
    return k.SetDWordValue("MTU", uint32(mtu))*/
    cmd := exec.Command("netsh", "interface", "ipv4", "set", "subinterface", 
        "\"" + t.humanReadableName + "\"", "mtu="+strconv.Itoa(mtu), "store=persistent")
    return cmd.Run()
}

func (t *tunWindows) Write(b []byte) (int, error) {
    if t.ready {
        return windows.Write(t.file, b)
    }
    return 0, errAdapterNotConfigured
}

func getHumanReadableName(guid string) (name string, err error) {
    var key registry.Key
    key, err = registry.OpenKey(
        registry.LOCAL_MACHINE, 
        "SYSTEM\\CurrentControlSet\\Control\\Network\\{4D36E972-E325-11CE-BFC1-08002BE10318}\\" + guid + "\\Connection", 
        registry.READ | registry.QUERY_VALUE)
    if err != nil {
        return
    }
    name, _, err = key.GetStringValue("Name")
    return
}
func getAdapterGUID() (guid string, err error) {
    var rootKey registry.Key
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

func ctlCode(DeviceType, Function, Method, Access uint32) uint32 {
    return ((DeviceType << 16) | (Access << 14) | (Function << 2) | Method);
}
