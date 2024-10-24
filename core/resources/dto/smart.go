package dto

type Smartctl struct {
	Version              []int    `json:"version"`
	SvnRevision          string   `json:"svn_revision"`
	PlatformInfo         string   `json:"platform_info"`
	BuildInfo            string   `json:"build_info"`
	Argv                 []string `json:"argv"`
	DriveDatabaseVersion struct {
		String string `json:"string"`
	} `json:"drive_database_version"`
	ExitStatus int `json:"exit_status"`
}

type LocalTime struct {
	TimeT   int    `json:"time_t"`
	Asctime string `json:"asctime"`
}

type Device struct {
	Name     string `json:"name"`
	InfoName string `json:"info_name"`
	Type     string `json:"type"`
	Protocol string `json:"protocol"`
}

type Flags struct {
	Value         int    `json:"value"`
	String        string `json:"string"`
	Prefailure    bool   `json:"prefailure"`
	UpdatedOnline bool   `json:"updated_online"`
	Performance   bool   `json:"performance"`
	ErrorRate     bool   `json:"error_rate"`
	EventCount    bool   `json:"event_count"`
	AutoKeep      bool   `json:"auto_keep"`
}

type Raw struct {
	Value  int    `json:"value"`
	String string `json:"string"`
}

type Attribute struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Value      int    `json:"value"`
	Worst      int    `json:"worst"`
	Thresh     int    `json:"thresh"`
	WhenFailed string `json:"when_failed"`
	Flags      Flags  `json:"flags"`
	Raw        Raw    `json:"raw"`
}

type AtaSmartAttributes struct {
	Revision int         `json:"revision"`
	Table    []Attribute `json:"table"`
}

type PowerOnTime struct {
	Hours int `json:"hours"`
}

type Temperature struct {
	Current int `json:"current"`
}

type WWN struct {
	Naa int `json:"naa"`
	Oui int `json:"oui"`
	ID  int `json:"id"`
}

type UserCapacity struct {
	Blocks uint64 `json:"blocks"`
	Bytes  uint64 `json:"bytes"`
}

type FormFactor struct {
	AtaValue int    `json:"ata_value"`
	Name     string `json:"name"`
}

type AtaVersion struct {
	String     string `json:"string"`
	MajorValue int    `json:"major_value"`
	MinorValue int    `json:"minor_value"`
}

type SataVersion struct {
	String string `json:"string"`
	Value  int    `json:"value"`
}

type InterfaceSpeed struct {
	Max struct {
		SataValue      int    `json:"sata_value"`
		String         string `json:"string"`
		UnitsPerSecond int    `json:"units_per_second"`
		BitsPerUnit    int    `json:"bits_per_unit"`
	} `json:"max"`
	Current struct {
		SataValue      int    `json:"sata_value"`
		String         string `json:"string"`
		UnitsPerSecond int    `json:"units_per_second"`
		BitsPerUnit    int    `json:"bits_per_unit"`
	} `json:"current"`
}

type SmartSupport struct {
	Available bool `json:"available"`
	Enabled   bool `json:"enabled"`
}

type SmartStatus struct {
	Passed bool `json:"passed"`
}

type AtaSmartErrorLog struct {
	Summary struct {
		Revision int `json:"revision"`
		Count    int `json:"count"`
	} `json:"summary"`
}

type SmartctlOutput struct {
	JSONFormatVersion []int        `json:"json_format_version"`
	Smartctl          Smartctl     `json:"smartctl"`
	LocalTime         LocalTime    `json:"local_time"`
	Device            Device       `json:"device"`
	ModelFamily       string       `json:"model_family"`
	ModelName         string       `json:"model_name"`
	SerialNumber      string       `json:"serial_number"`
	WWN               WWN          `json:"wwn"`
	FirmwareVersion   string       `json:"firmware_version"`
	UserCapacity      UserCapacity `json:"user_capacity"`
	LogicalBlockSize  int          `json:"logical_block_size"`
	PhysicalBlockSize int          `json:"physical_block_size"`
	RotationRate      int          `json:"rotation_rate"`
	FormFactor        FormFactor   `json:"form_factor"`
	Trim              struct {
		Supported bool `json:"supported"`
	} `json:"trim"`
	InSmartctlDatabase bool               `json:"in_smartctl_database"`
	AtaVersion         AtaVersion         `json:"ata_version"`
	SataVersion        SataVersion        `json:"sata_version"`
	InterfaceSpeed     InterfaceSpeed     `json:"interface_speed"`
	SmartSupport       SmartSupport       `json:"smart_support"`
	SmartStatus        SmartStatus        `json:"smart_status"`
	AtaSmartAttributes AtaSmartAttributes `json:"ata_smart_attributes"`
	PowerOnTime        PowerOnTime        `json:"power_on_time"`
	PowerCycleCount    int                `json:"power_cycle_count"`
	Temperature        Temperature        `json:"temperature"`
	AtaSmartErrorLog   AtaSmartErrorLog   `json:"ata_smart_error_log"`
}
