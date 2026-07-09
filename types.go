package main

import "time"

type ContextMeta struct {
	Name             string    `yaml:"name"`
	AzureConfigDir   string    `yaml:"azure_config_dir"`
	SubscriptionID   string    `yaml:"subscription_id,omitempty"`
	SubscriptionName string    `yaml:"subscription_name,omitempty"`
	TenantID         string    `yaml:"tenant_id,omitempty"`
	TenantName       string    `yaml:"tenant_name,omitempty"`
	User             string    `yaml:"user,omitempty"`
	CreatedAt        time.Time `yaml:"created_at"`
	UpdatedAt        time.Time `yaml:"updated_at"`
	LastUsedAt       time.Time `yaml:"last_used_at,omitempty"`
}

type AzAccount struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	TenantID string `json:"tenantId"`
	User     struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"user"`
}

type AzTenant struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenantId"`
	DisplayName string `json:"displayName"`
}
