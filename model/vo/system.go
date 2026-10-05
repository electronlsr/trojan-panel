package vo

type SystemVo struct {
	Id                          uint `json:"id" redis:"id"`
	RegisterEnable              uint `json:"registerEnable" redis:"registerEnable"`
	RegisterQuota               int  `json:"registerQuota" redis:"registerQuota"`
	RegisterExpireDays          uint `json:"registerExpireDays" redis:"registerExpireDays"`
	ResetDownloadAndUploadMonth uint `json:"resetDownloadAndUploadMonth" redis:"resetDownloadAndUploadMonth"`
	TrafficRankEnable           uint `json:"trafficRankEnable" redis:"trafficRankEnable"`
	CaptchaEnable               uint `json:"captchaEnable" redis:"captchaEnable"`

	ExpireWarnEnable uint   `json:"expireWarnEnable" redis:"expireWarnEnable"`
	ExpireWarnDay    uint   `json:"expireWarnDay" redis:"expireWarnDay"`
	EmailEnable      uint   `json:"emailEnable"`
	EmailHost        string `json:"emailHost" redis:"emailHost"`
	EmailPort        uint   `json:"emailPort" redis:"emailPort"`
	EmailUsername    string `json:"emailUsername" redis:"emailUsername"`
	EmailPassword    string `json:"emailPassword" redis:"emailPassword"`

	SystemName   string `json:"systemName" redis:"systemName"`
	ClashRule    string `json:"clashRule" redis:"clashRule"`
	XrayTemplate string `json:"xrayTemplate" redis:"xrayTemplate"`
}

type SettingVo struct {
	Version            string `json:"version"`
	RegisterEnable     uint   `json:"registerEnable"`
	RegisterQuota      int    `json:"registerQuota"`
	RegisterExpireDays uint   `json:"registerExpireDays"`
	TrafficRankEnable  uint   `json:"trafficRankEnable"`
	CaptchaEnable      uint   `json:"captchaEnable" redis:"captchaEnable"`
	EmailEnable        uint   `json:"emailEnable"`
	SystemName         string `json:"systemName" redis:"systemName"`
}
