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

type SmartctlOutput struct {
	JsonFormatVersion  []int              `json:"json_format_version"`
	Smartctl           Smartctl           `json:"smartctl"`
	LocalTime          LocalTime          `json:"local_time"`
	Device             Device             `json:"device"`
	AtaSmartAttributes AtaSmartAttributes `json:"ata_smart_attributes"`
	PowerOnTime        PowerOnTime        `json:"power_on_time"`
	PowerCycleCount    int                `json:"power_cycle_count"`
	Temperature        Temperature        `json:"temperature"`
}
