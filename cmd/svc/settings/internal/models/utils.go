package models

import "bytes"

func FlattenConfigKey(key *ConfigKey) string {
	var b bytes.Buffer
	b.WriteString(key.Key)
	if key.Subkey != "" {
		b.WriteString(".")
		b.WriteString(key.Subkey)
	}
	return b.String()
}
