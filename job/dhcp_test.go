package job

import (
	"bytes"
	"fmt"
	"net"
	"testing"

	"github.com/go-logr/logr"
	dhcp4 "github.com/packethost/dhcp4-go"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/client/standalone"
)

func TestSetPXEFilename(t *testing.T) {
	publicFQDN := "boots-testing.packet.net"

	setPXEFilenameTests := []struct {
		name       string
		hState     string
		id         string
		iState     string
		slug       string
		plan       string
		allowPXE   bool
		packet     bool
		arm        bool
		uefi       bool
		httpClient bool
		filename   string
	}{
		{
			name:   "just in_use",
			hState: "in_use",
		},
		{
			name:   "no instance state",
			hState: "in_use", id: "$instance_id", iState: "",
		},
		{
			name:   "instance not active",
			hState: "in_use", id: "$instance_id", iState: "not_active",
		},
		{
			name:   "instance active",
			hState: "in_use", id: "$instance_id", iState: "active",
		},
		{
			name:   "active not custom ipxe",
			hState: "in_use", id: "$instance_id", iState: "active", slug: "not_custom_ipxe",
		},
		{
			name:   "active custom ipxe",
			hState: "in_use", id: "$instance_id", iState: "active", slug: "custom_ipxe",
			filename: "undionly.kpxe",
		},
		{
			name:   "active custom ipxe with allow pxe",
			hState: "in_use", id: "$instance_id", iState: "active", allowPXE: true,
			filename: "undionly.kpxe",
		},
		{
			name: "arm",
			arm:  true, filename: "snp.efi",
		},
		{
			name: "x86 uefi",
			uefi: true, filename: "ipxe.efi",
		},
		{
			name: "x86 uefi http client",
			uefi: true, allowPXE: true, httpClient: true,
			filename: "http://" + publicFQDN + "/ipxe/ipxe.efi",
		},
		{
			name:     "all defaults",
			filename: "undionly.kpxe",
		},
		{
			name:   "packet iPXE",
			packet: true, filename: "nonexistent",
		},
		{
			name:   "packet iPXE PXE allowed",
			packet: true, id: "$instance_id", allowPXE: true, filename: "http://" + publicFQDN + "/auto.ipxe",
		},
	}

	for _, tt := range setPXEFilenameTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("%+v", tt)

			if tt.plan == "" {
				tt.plan = "0"
			}

			instance := &client.Instance{
				ID:       tt.id,
				State:    client.InstanceState(tt.iState),
				AllowPXE: tt.allowPXE,
				OS: &client.OperatingSystem{
					OsSlug: tt.slug,
				},
				OSV: &client.OperatingSystem{
					OsSlug: tt.slug,
				},
			}
			j := Job{
				Logger: logr.Discard(),
				hardware: &standalone.HardwareStandalone{
					ID: "$hardware_id",
					Metadata: client.Metadata{
						State: client.HardwareState(tt.hState),
						Facility: client.Facility{
							PlanSlug: "baremetal_" + tt.plan,
						},
						Instance: instance,
					},
				},
				instance:     instance,
				NextServer:   net.IPv4(127, 0, 0, 1),
				IpxeBaseURL:  publicFQDN + "/ipxe",
				BootsBaseURL: publicFQDN,
			}
			rep := dhcp4.NewPacket(42)
			j.setPXEFilename(&rep, tt.packet, tt.arm, tt.uefi, tt.httpClient)
			filename := string(bytes.TrimRight(rep.File(), "\x00"))

			if tt.filename != filename {
				t.Fatalf("unexpected filename want:%q, got:%q", tt.filename, filename)
			}
		})
	}
}

func TestAllowPXE(t *testing.T) {
	for _, tt := range []struct {
		want     bool
		hw       bool
		instance bool
		iid      string
	}{
		{want: true, hw: true},
		{want: false, hw: false, instance: true},
		{want: true, hw: false, instance: true, iid: "id"},
		{want: false, hw: false, instance: false, iid: "id"},
	} {
		name := fmt.Sprintf("want=%t, hardware=%t, instance=%t, instance_id=%s", tt.want, tt.hw, tt.instance, tt.iid)
		t.Run(name, func(t *testing.T) {
			j := Job{
				hardware: &standalone.HardwareStandalone{
					ID: "$hardware_id",
					Metadata: client.Metadata{
						Instance: &client.Instance{
							AllowPXE: tt.hw,
						},
					},
					Network: client.Network{
						Interfaces: []client.NetworkInterface{
							{
								Netboot: client.Netboot{
									AllowPXE: tt.hw,
								},
							},
						},
					},
				},
				instance: &client.Instance{
					ID:       tt.iid,
					AllowPXE: tt.instance,
				},
			}
			got := j.AllowPXE()
			if got != tt.want {
				t.Fatalf("unexpected return, want: %t, got %t", tt.want, got)
			}
		})
	}
}
