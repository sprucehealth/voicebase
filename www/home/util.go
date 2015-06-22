package home

import "github.com/sprucehealth/backend/common"

type statesByAbbr []*common.State

func (s statesByAbbr) Len() int           { return len(s) }
func (s statesByAbbr) Less(a, b int) bool { return s[a].Abbreviation < s[b].Abbreviation }
func (s statesByAbbr) Swap(a, b int)      { s[a], s[b] = s[b], s[a] }

type statesByName []*common.State

func (s statesByName) Len() int           { return len(s) }
func (s statesByName) Less(a, b int) bool { return s[a].Name < s[b].Name }
func (s statesByName) Swap(a, b int)      { s[a], s[b] = s[b], s[a] }
