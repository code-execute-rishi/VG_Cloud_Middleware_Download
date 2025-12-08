package models

type ZerotierConfigResponse struct {
    ZerotierIP  string   `json:"zerotier_ip"`
    SSHCommand  string   `json:"ssh_command"`
}

type ZerotierConfig struct {
    ZerotierIP string
}

type ZerotierMemberResponse struct {
    Config struct {
        IPAssignments []string `json:"ipAssignments"`
        Authorized    bool     `json:"authorized"`
    } `json:"config"`
}