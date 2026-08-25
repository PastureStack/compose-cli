package config

// LegacyLoadBalancerConfig preserves only the Rancher 1.6 Compose fields that
// are still accepted at the compatibility boundary. Keeping these data types
// locally avoids importing the abandoned v1 generated API client.
type LegacyLoadBalancerConfig struct {
	HaproxyConfig            *LegacyHaproxyConfig                      `json:"haproxyConfig,omitempty" yaml:"haproxy_config,omitempty"`
	LbCookieStickinessPolicy *LegacyLoadBalancerCookieStickinessPolicy `json:"lbCookieStickinessPolicy,omitempty" yaml:"lb_cookie_stickiness_policy,omitempty"`
}

type LegacyHaproxyConfig struct {
	Defaults string `json:"defaults,omitempty" yaml:"defaults,omitempty"`
	Global   string `json:"global,omitempty" yaml:"global,omitempty"`
}

type LegacyLoadBalancerCookieStickinessPolicy struct {
	Cookie   string `json:"cookie,omitempty" yaml:"cookie,omitempty"`
	Domain   string `json:"domain,omitempty" yaml:"domain,omitempty"`
	Indirect bool   `json:"indirect,omitempty" yaml:"indirect,omitempty"`
	Mode     string `json:"mode,omitempty" yaml:"mode,omitempty"`
	Name     string `json:"name,omitempty" yaml:"name,omitempty"`
	Nocache  bool   `json:"nocache,omitempty" yaml:"nocache,omitempty"`
	Postonly bool   `json:"postonly,omitempty" yaml:"postonly,omitempty"`
}
