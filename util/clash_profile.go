package util

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"reflect"
	"strings"
	"trojan-panel/model/constant"

	"gopkg.in/yaml.v3"
)

const ClashVergeTarget = "clash-verge"

// BuildClashProfile merges generated account nodes with the administrator's
// template without dropping unrecognized Mihomo fields such as dns or providers.
// The legacy target keeps the administrator's routing policy. The Verge target
// adds private/CN bypass rules and independently updating rule providers.
func BuildClashProfile(generated []byte, template, target string) ([]byte, error) {
	profile, err := decodeClashMapping(generated)
	if err != nil {
		return nil, fmt.Errorf("generated Clash profile: %w", err)
	}
	if proxies, ok := profile["proxies"].([]interface{}); !ok || len(proxies) == 0 {
		return nil, errors.New("subscription has no supported proxy nodes")
	}
	config, err := decodeClashMapping([]byte(template))
	if err != nil {
		return nil, fmt.Errorf("Clash template: %w", err)
	}
	for key, value := range config {
		if _, exists := profile[key]; exists {
			return nil, fmt.Errorf("Clash template duplicates generated field %q", key)
		}
		profile[key] = value
	}
	rules, err := clashRules(profile["rules"])
	if err != nil {
		return nil, err
	}
	// Older installed templates contain this invalid upstream rule. Remove only
	// this exact malformed rule; valid custom GEOIP rules are left untouched.
	rules = withoutBrokenLANRule(rules)
	profile["rules"] = rules
	if target != ClashVergeTarget {
		return yaml.Marshal(profile)
	}

	var stock map[string]interface{}
	if err := yaml.Unmarshal([]byte(constant.ClashRules), &stock); err != nil {
		return nil, err
	}
	stockRules, _ := clashRules(stock["rules"])
	// Match only the stock routing sections, allowing extra custom DNS/settings.
	// Do not rewrite an administrator's customized routing/provider sections.
	if reflect.DeepEqual(rules, stockRules) && reflect.DeepEqual(profile["rule-providers"], stock["rule-providers"]) {
		rules = nil
		delete(profile, "rule-providers")
	}
	providers := map[string]interface{}{}
	if existing, ok := profile["rule-providers"]; ok && existing != nil {
		var valid bool
		providers, valid = existing.(map[string]interface{})
		if !valid {
			return nil, errors.New("rule-providers must be a mapping")
		}
	}

	// Reserve unique names/paths so custom providers are never overwritten.
	prefix := "tp-verge"
	for suffix := 2; providerPrefixUsed(providers, prefix); suffix++ {
		prefix = fmt.Sprintf("tp-verge-%d", suffix)
	}
	for _, source := range []struct{ name, behavior, path string }{
		{"private-domain", "domain", "geosite/private"},
		{"private-ip", "ipcidr", "geoip/private"},
		{"cn-domain", "domain", "geosite/cn"},
		{"cn-ip", "ipcidr", "geoip/cn"},
	} {
		name := prefix + "-" + source.name
		providers[name] = map[string]interface{}{
			"type": "http", "behavior": source.behavior, "format": "mrs",
			"url":  "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/" + source.path + ".mrs",
			"path": "./ruleset/" + name + ".mrs", "interval": 86400,
			"proxy": "PROXY",
		}
	}
	profile["rule-providers"] = providers
	profile["mode"] = "rule"

	// Static LAN rules still work before providers have downloaded. no-resolve
	// avoids needless DNS for literal/private addresses; the later private-ip
	// rule also checks domains that resolve to private IPs.
	merged := []string{
		"DOMAIN,localhost,DIRECT",
		"DOMAIN-SUFFIX,localhost,DIRECT",
		"DOMAIN-SUFFIX,local,DIRECT",
		"DOMAIN-SUFFIX,lan,DIRECT",
		"DOMAIN-SUFFIX,home.arpa,DIRECT",
		"IP-CIDR,0.0.0.0/8,DIRECT,no-resolve",
		"IP-CIDR,10.0.0.0/8,DIRECT,no-resolve",
		"IP-CIDR,100.64.0.0/10,DIRECT,no-resolve",
		"IP-CIDR,127.0.0.0/8,DIRECT,no-resolve",
		"IP-CIDR,169.254.0.0/16,DIRECT,no-resolve",
		"IP-CIDR,172.16.0.0/12,DIRECT,no-resolve",
		"IP-CIDR,192.168.0.0/16,DIRECT,no-resolve",
		"IP-CIDR,224.0.0.0/4,DIRECT,no-resolve",
		"IP-CIDR,255.255.255.255/32,DIRECT,no-resolve",
		"IP-CIDR6,::/127,DIRECT,no-resolve",
		"IP-CIDR6,fc00::/7,DIRECT,no-resolve",
		"IP-CIDR6,fe80::/10,DIRECT,no-resolve",
		"IP-CIDR6,ff00::/8,DIRECT,no-resolve",
		"RULE-SET," + prefix + "-private-domain,DIRECT",
		"RULE-SET," + prefix + "-private-ip,DIRECT,no-resolve",
	}
	var terminal []string
	for i, rule := range rules {
		kind := strings.TrimSpace(strings.SplitN(rule, ",", 2)[0])
		if kind == "MATCH" || kind == "FINAL" {
			terminal = rules[i:]
			break
		}
		merged = append(merged, rule)
	}
	merged = append(merged,
		"RULE-SET,"+prefix+"-cn-domain,DIRECT",
		"RULE-SET,"+prefix+"-private-ip,DIRECT",
		"RULE-SET,"+prefix+"-cn-ip,DIRECT",
	)
	if len(terminal) == 0 {
		terminal = []string{"MATCH,PROXY"}
	}
	profile["rules"] = append(merged, terminal...)
	return yaml.Marshal(profile)
}

func decodeClashMapping(data []byte) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&result); err != nil && err != io.EOF {
		return nil, err
	}
	if result == nil {
		result = map[string]interface{}{}
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("expected one YAML document")
	}
	return result, nil
}

func clashRules(value interface{}) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	list, ok := value.([]interface{})
	if !ok {
		return nil, errors.New("rules must be a YAML list")
	}
	rules := make([]string, 0, len(list))
	for _, value := range list {
		rule, ok := value.(string)
		if !ok || strings.TrimSpace(rule) == "" {
			return nil, errors.New("each rule must be a nonempty string")
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func withoutBrokenLANRule(rules []string) []string {
	result := make([]string, 0, len(rules))
	for _, rule := range rules {
		if strings.ReplaceAll(rule, " ", "") != "GEOIP,,DIRECT" {
			result = append(result, rule)
		}
	}
	return result
}

func providerPrefixUsed(providers map[string]interface{}, prefix string) bool {
	for _, name := range []string{"private-domain", "private-ip", "cn-domain", "cn-ip"} {
		if _, exists := providers[prefix+"-"+name]; exists {
			return true
		}
		// A custom provider can have a different name but the same cache path.
		for _, existing := range providers {
			if provider, ok := existing.(map[string]interface{}); ok {
				if cachePath, ok := provider["path"].(string); ok && strings.EqualFold(path.Clean(strings.ReplaceAll(cachePath, "\\", "/")), "ruleset/"+prefix+"-"+name+".mrs") {
					return true
				}
			}
		}
	}
	return false
}
