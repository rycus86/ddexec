package config

import (
	"regexp"
	"strings"
)

func (c *Configuration) GetImage() string {
	if c.Image != "" {
		return c.Image
	}

	return strings.TrimSuffix(
		strings.TrimSuffix(
			strings.TrimSuffix(
				c.Filename, ".yaml",
			), ".yml",
		), ".dapp",
	)
}

func (c *Configuration) GetName() string {
	if c.Name != "" {
		return c.Name
	}

	return regexp.MustCompile("(?:.*/)?(.+?)(?::.*)?").ReplaceAllString(c.GetImage(), "$1")
}
