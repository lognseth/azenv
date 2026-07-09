package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
)

func azAccount(cfg string) (*AzAccount, error) {
	c := exec.Command("az", "account", "show", "-o", "json")
	c.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+cfg)
	out, err := c.Output()
	if err != nil {
		return nil, err
	}
	var acc AzAccount
	if err := json.Unmarshal(out, &acc); err != nil {
		return nil, err
	}
	return &acc, nil
}

func azTenants(cfg string) ([]AzTenant, error) {
	c := exec.Command("az", "account", "tenant", "list", "-o", "json")
	c.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+cfg)
	out, err := c.Output()
	if err != nil {
		return nil, err
	}
	var tenants []AzTenant
	if err := json.Unmarshal(out, &tenants); err != nil {
		return nil, err
	}
	return tenants, nil
}

func refreshAzureMetadata(meta *ContextMeta) error {
	acc, err := azAccount(meta.AzureConfigDir)
	if err != nil {
		return err
	}
	meta.SubscriptionID = acc.ID
	meta.SubscriptionName = acc.Name
	meta.TenantID = acc.TenantID
	meta.User = acc.User.Name
	if tenants, err := azTenants(meta.AzureConfigDir); err == nil {
		for _, t := range tenants {
			id := t.TenantID
			if id == "" {
				id = t.ID
			}
			if id == meta.TenantID {
				meta.TenantName = t.DisplayName
				break
			}
		}
	}
	return saveMeta(meta)
}

func refreshAzureMetadataBestEffort(meta *ContextMeta) {
	_ = refreshAzureMetadata(meta)
}

func runAzWithConfig(cfg string, args ...string) error {
	if _, err := exec.LookPath("az"); err != nil {
		return errors.New("Azure CLI not found on PATH")
	}
	c := exec.Command("az", args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+cfg)
	return c.Run()
}
