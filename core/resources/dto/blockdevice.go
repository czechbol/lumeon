package dto

type BlockDevice struct {
	Name       string       `json:"name"`
	Size       uint64       `json:"size"`
	Type       string       `json:"type"`
	MountPoint string       `json:"mountpoint,omitempty"`
	Children   []BlockChild `json:"children,omitempty"`
}

type BlockChild struct {
	Name       string `json:"name"`
	Size       uint64 `json:"size"`
	Type       string `json:"type"`
	MountPoint string `json:"mountpoint,omitempty"`
}

type BlockDevices struct {
	BlockDevices []*BlockDevice `json:"blockdevices"`
}
