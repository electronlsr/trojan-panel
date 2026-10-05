package constant

// 系统设置
const (
	// SystemName 系统名称
	SystemName string = "trojan-panel"
	// WebFilePath 伪装站点文件夹路径
	WebFilePath string = "webfile"
	// WebFileName 伪装站点文件名称
	WebFileName string = "webfile.zip"
	// LogPath 日志文件夹路径
	LogPath string = "logs"
	// ConfigPath 配置文件夹路径
	ConfigPath   string = "config"
	TemplatePath string = "config/template"
	// LogoImagePath 系统logo文件路径
	LogoImagePath string = "config/template/logo.png"
	LogoImageUrl  string = "https://raw.githubusercontent.com/trojanpanel/trojanpanel.github.io/main/docs/logo.png"
	// ConfigFilePath 配置文件路径
	ConfigFilePath string = "config/config.ini"
	// RbacModelFilePath rbac配置文件路径
	RbacModelFilePath string = "config/rbac_model.conf"
	// ClashRuleFilePath Clash规则默认模板
	ClashRuleFilePath string = "config/template/template-clash-rule.yaml"
	// XrayTemplateFilePath Xray模板
	XrayTemplateFilePath string = "config/template/template-xray.json"

	ExportPath               string = "config/export"
	ExportAccountTemplate    string = "config/export/AccountTemplate.json"
	ExportNodeServerTemplate string = "config/export/NodeServerTemplate.json"

	TrojanPanelCertFilePath string = "/tpdata/trojan-panel/cert/"
	TrojanPanelCrtFile      string = "trojan-panel.crt"
	TrojanPanelKeyFile      string = "trojan-panel.key"

	TrojanPanelVersion = "v3.0.0"
)
