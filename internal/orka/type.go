// Copyright 2020 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package orka

// Config configures a virtual machine.
type Config struct {
	Name  string `json:"orka_vm_name"`
	Image string `json:"orka_base_image"`
	CPU   int    `json:"orka_cpu_core"`
	VCPU  int    `json:"vcpu_count"`
}

type (
	// Response provides the API response.
	Response struct {
		Message string   `json:"message"`
		Errors  []*Error `json:"errors"`
	}

	// CreateRequest provides the create API request.
	//
	//     {
	// 	    "orka_vm_name": "myorkavm",
	// 	    "orka_base_image": "myStorage.img",
	// 	    "orka_image": "myorkavm",
	// 	    "orka_cpu_core": 6,
	// 	    "vcpu_count": 6,
	// 	    "iso_image": "Mojave.iso"
	//     }
	//
	// TODO(bradrydzewski) model the full json structure (above) and
	// replace and remove the Config struct with this struct.
	CreateRequest struct {
		Name  string `json:"orka_vm_name"`
		Image string `json:"orka_base_image"`
		CPU   int    `json:"orka_cpu_core"`
		VCPU  int    `json:"vcpu_count"`
	}

	// DeployResponse provides the deployment API response.
	DeployResponse struct {
		Response
		RAM             string        `json:"ram"`
		VCPU            string        `json:"vcpu"`
		HostCPU         string        `json:"host_cpu"`
		IP              string        `json:"ip"`
		SSHPort         string        `json:"ssh_port"`
		ScreenSharePort string        `json:"screen_share_port"`
		VMID            string        `json:"vm_id"`
		PortWarnings    []interface{} `json:"port_warnings"`
		VncPort         string        `json:"vnc_port"`
	}

	// StatusResponse provides the status API response.
	StatusResponse struct {
		Response
		VirtualMachineResources []struct {
			VirtualMachineName string `json:"virtual_machine_name"`
			VMDeploymentStatus string `json:"vm_deployment_status"`
			Status             []struct {
				Owner                 string `json:"owner"`
				VirtualMachineName    string `json:"virtual_machine_name"`
				VirtualMachineID      string `json:"virtual_machine_id"`
				NodeLocation          string `json:"node_location"`
				NodeStatus            string `json:"node_status"`
				VirtualMachineIP      string `json:"virtual_machine_ip"`
				VncPort               string `json:"vnc_port"`
				ScreenSharingPort     string `json:"screen_sharing_port"`
				SSHPort               string `json:"ssh_port"`
				CPU                   int    `json:"cpu"`
				Vcpu                  int    `json:"vcpu"`
				RAM                   string `json:"RAM"`
				BaseImage             string `json:"base_image"`
				Image                 string `json:"image"`
				ConfigurationTemplate string `json:"configuration_template"`
				VMStatus              string `json:"vm_status"`
				ReservedPorts         []struct {
					HostPort  int    `json:"host_port"`
					GuestPort int    `json:"guest_port"`
					Protocol  string `json:"protocol"`
				} `json:"reserved_ports"`
			} `json:"status"`
		} `json:"virtual_machine_resources"`
	}

	// TokenResponse provides the token API response.
	TokenResponse struct {
		Response
		Authenticated  bool   `json:"authenticated"`
		IsTokenRevoked bool   `json:"is_token_revoked"`
		Email          string `json:"email"`
	}
)

// Error represents an API error.
type Error struct {
	Message string `json:"message"`
}

// Error returns the error message.
func (e *Error) Error() string {
	return e.Message
}
