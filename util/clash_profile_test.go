package util

import (
	"reflect"
	"strings"
	"testing"
	"trojan-panel/model/constant"

	"gopkg.in/yaml.v3"
)

const testGeneratedClash = `proxies:
  - {name: Test-Trojan, type: trojan, server: example.com, port: 443, password: test-only}
proxy-groups:
  - {name: PROXY, type: select, proxies: [Test-Trojan]}
`

func readProfile(t *testing.T, template, target string) map[string]interface{} {
	t.Helper()
	data, err := BuildClashProfile([]byte(testGeneratedClash), template, target)
	if err != nil {
		t.Fatal(err)
	}
	var profile map[string]interface{}
	if err := yaml.Unmarshal(data, &profile); err != nil {
		t.Fatal(err)
	}
	return profile
}

func TestClashProfileRepairsPersistedEmptyGEOIP(t *testing.T) {
	// This is the exact invalid rule reported by Clash Verge, in an already
	// persisted template from a pre-upgrade installation (not the new constant).
	oldTemplate := strings.Replace(constant.ClashRules, "  - GEOIP,CN,DIRECT", "  - GEOIP,,DIRECT\n  - GEOIP,CN,DIRECT", 1)
	for _, target := range []string{"", ClashVergeTarget} {
		t.Run(target, func(t *testing.T) {
			profile := readProfile(t, oldTemplate, target)
			rules, _ := clashRules(profile["rules"])
			for _, rule := range rules {
				if rule == "GEOIP,,DIRECT" {
					t.Fatal("persisted invalid rule survived")
				}
			}
			if target == "" && !reflect.DeepEqual(rules, mustStockRules(t)) {
				t.Fatal("legacy custom routing unexpectedly changed")
			}
			providers := profile["rule-providers"].(map[string]interface{})
			if target == ClashVergeTarget && len(providers) != 4 {
				t.Fatalf("stock provider migration failed: %d providers", len(providers))
			}
		})
	}
}

func mustStockRules(t *testing.T) []string {
	t.Helper()
	var stock map[string]interface{}
	_ = yaml.Unmarshal([]byte(constant.ClashRules), &stock)
	rules, err := clashRules(stock["rules"])
	if err != nil {
		t.Fatal(err)
	}
	return rules
}

func TestClashVergePreservesCustomSettingsAndOrder(t *testing.T) {
	template := `dns:
  enable: true
  nameserver: [system]
  nameserver-policy:
    '+.corp.example': 192.168.1.1
ipv6: false
mixed-port: 7899
rule-providers:
  custom:
    type: inline
    behavior: classical
    payload: ["DOMAIN,custom.example"]
rules:
  - DOMAIN,example.cn,PROXY
  - RULE-SET,custom,REJECT
  - GEOIP,,DIRECT
  - MATCH,DIRECT
`
	profile := readProfile(t, template, ClashVergeTarget)
	var original map[string]interface{}
	_ = yaml.Unmarshal([]byte(template), &original)
	for _, key := range []string{"dns", "ipv6", "mixed-port"} {
		if !reflect.DeepEqual(profile[key], original[key]) {
			t.Fatalf("custom %s changed", key)
		}
	}
	if profile["mode"] != "rule" {
		t.Fatal("expected rule mode")
	}
	if _, ok := profile["tun"]; ok {
		t.Fatal("TUN must not be enabled implicitly")
	}
	providers := profile["rule-providers"].(map[string]interface{})
	if !reflect.DeepEqual(providers["custom"], original["rule-providers"].(map[string]interface{})["custom"]) {
		t.Fatal("custom provider changed")
	}
	rules, _ := clashRules(profile["rules"])
	positions := map[string]int{}
	for i, rule := range rules {
		positions[rule] = i
	}
	if positions["IP-CIDR,192.168.0.0/16,DIRECT,no-resolve"] >= positions["DOMAIN,example.cn,PROXY"] ||
		positions["DOMAIN,example.cn,PROXY"] >= positions["RULE-SET,tp-verge-cn-domain,DIRECT"] {
		t.Fatal("expected LAN -> custom -> CN priority")
	}
	if rules[len(rules)-1] != "MATCH,DIRECT" {
		t.Fatal("custom terminal policy not preserved")
	}
}

func TestClashVergeStockWithDNS(t *testing.T) {
	profile := readProfile(t, constant.ClashRules+"\ndns:\n  nameserver: [system]\n", ClashVergeTarget)
	if len(profile["rule-providers"].(map[string]interface{})) != 4 || profile["dns"] == nil {
		t.Fatal("stock routing with custom DNS must migrate, preserving DNS")
	}
}

func TestClashVergeDefaultProvidersAndFallback(t *testing.T) {
	profile := readProfile(t, "", ClashVergeTarget)
	providers := profile["rule-providers"].(map[string]interface{})
	for _, entry := range []struct{ name, behavior, path string }{
		{"private-domain", "domain", "geosite/private"},
		{"private-ip", "ipcidr", "geoip/private"},
		{"cn-domain", "domain", "geosite/cn"},
		{"cn-ip", "ipcidr", "geoip/cn"},
	} {
		provider := providers["tp-verge-"+entry.name].(map[string]interface{})
		if provider["behavior"] != entry.behavior || provider["format"] != "mrs" || provider["interval"] != 86400 || provider["proxy"] != "PROXY" {
			t.Fatalf("invalid provider %s: %v", entry.name, provider)
		}
		if provider["url"] != "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/"+entry.path+".mrs" {
			t.Fatal("wrong provider URL")
		}
	}
	rules, _ := clashRules(profile["rules"])
	if rules[len(rules)-1] != "MATCH,PROXY" {
		t.Fatal("unmatched traffic must default to PROXY")
	}
	for _, cidr := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8", "100.64.0.0/10", "169.254.0.0/16", "::/127", "fc00::/7", "fe80::/10", "ff00::/8"} {
		if !strings.Contains(strings.Join(rules, "\n"), ","+cidr+",DIRECT,no-resolve") {
			t.Fatalf("missing LAN fallback %s", cidr)
		}
	}
}

func TestClashVergeProviderCollision(t *testing.T) {
	for _, custom := range []string{
		"tp-verge-cn-domain: {type: inline, behavior: domain, payload: ['+.example.com']}",
		"mine: {type: http, behavior: domain, path: ./ruleset/tp-verge-cn-domain.mrs, url: https://example.com/rules}",
		"mine: {type: http, behavior: domain, path: ruleset/tp-verge-cn-domain.mrs, url: https://example.com/rules}",
		"mine: {type: http, behavior: domain, path: RULESET/TP-VERGE-CN-DOMAIN.MRS, url: https://example.com/rules}",
		"mine: {type: http, behavior: domain, path: ./ruleset/../ruleset/tp-verge-cn-domain.mrs, url: https://example.com/rules}",
	} {
		profile := readProfile(t, "rule-providers:\n  "+custom+"\nrules: [MATCH,PROXY]", ClashVergeTarget)
		providers := profile["rule-providers"].(map[string]interface{})
		if len(providers) != 5 || providers["tp-verge-2-cn-domain"] == nil {
			t.Fatal("custom provider name/path collision not avoided")
		}
	}
}

func TestClashProfileRejectsMalformedTemplates(t *testing.T) {
	for _, template := range []string{
		"rules: [MATCH,PROXY]\nrules: []", "rules: not-a-list", "rules: [1]",
		"rules: ['']", "rules: []\n---\nrules: []", "proxies: []",
		"proxy-groups: []", "rule-providers: []", "- invalid-root",
	} {
		if _, err := BuildClashProfile([]byte(testGeneratedClash), template, ClashVergeTarget); err == nil {
			t.Errorf("accepted malformed template %q", template)
		}
	}
}

func TestClashVergeKeepsUnreachableCustomTail(t *testing.T) {
	profile := readProfile(t, "rules:\n  - MATCH,DIRECT\n  - DOMAIN,baidu.com,REJECT\n", ClashVergeTarget)
	rules, _ := clashRules(profile["rules"])
	if rules[len(rules)-2] != "MATCH,DIRECT" || rules[len(rules)-1] != "DOMAIN,baidu.com,REJECT" {
		t.Fatal("rules after a custom catch-all must remain unreachable")
	}
}

func TestClashProfileRejectsEmptyProxyList(t *testing.T) {
	if _, err := BuildClashProfile([]byte("proxies: []\nproxy-groups: []"), constant.ClashRules, ClashVergeTarget); err == nil {
		t.Fatal("empty subscriptions must return an error rather than unimportable YAML")
	}
}
