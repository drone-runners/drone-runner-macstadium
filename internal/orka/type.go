// Copyright 2020 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package orka

// Config configures a virtual machine.
type Config struct {
	Name  string `json:"orka_vm_name"` 
	Image string `json:"orka_base_image"` // 90GCatalinaSSH.img
	CPU   int    `json:"orka_cpu_core"`   // 6
	VCPU  int    `json:"vcpu_count"`      // 6
}

type (
	// Response provides the API response.
	Response struct {
		Message string       `json:"message"`
		Errors []interface{} `json:"errors"`
	}

	// CreateRequest provides the create API request.
	CreateRequest struct {
		Name  string `json:"orka_vm_name"` 
		Image string `json:"orka_base_image"` // 90GCatalinaSSH.img
		CPU   int    `json:"orka_cpu_core"`   // 6
		VCPU  int    `json:"vcpu_count"`      // 6

		// {
		// 	"orka_vm_name": "myorkavm",
		// 	"orka_base_image": "myStorage.img",
		// 	"orka_image": "myorkavm",
		// 	"orka_cpu_core": 6,
		// 	"vcpu_count": 6,
		// 	"iso_image": "Mojave.iso"
		// }
	}

	// CreateResponse provides the create API response.
	CreateResponse struct {
		Message string       `json:"message"`
		Errors []interface{} `json:"errors"`
	}

	// DeleteResponse provides the delete API response.
	DeleteResponse struct {
		Message string       `json:"message"`
		Errors []interface{} `json:"errors"`
	}

	// DeployResponse provides the deployment API response.
	DeployResponse struct {
		Message         string        `json:"message"`
		Errors          []interface{} `json:"errors"`
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
		Message string `json:"message"`
		Errors                  []interface{} `json:"errors"`
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

	// TokenReponse provides the token API response.
	TokenReponse struct {
		Message string `json:"message"`
		Errors  []interface{} `json:"errors"`
		Authenticated bool  `json:"authenticated"`
		IsTokenRevoked bool `json:"is_token_revoked"`
		Email string `json:"email"`
	}
)
