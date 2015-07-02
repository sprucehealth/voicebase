package api

import (
	"sort"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
)

type byAccountGroupName []*common.AccountGroup

func (s byAccountGroupName) Len() int           { return len(s) }
func (s byAccountGroupName) Less(a, b int) bool { return s[a].Name < s[b].Name }
func (s byAccountGroupName) Swap(a, b int)      { s[a], s[b] = s[b], s[a] }

func (m *auth) AvailableAccountPermissions() ([]string, error) {
	perms := make([]string, 0, len(m.perms))
	for _, p := range m.perms {
		perms = append(perms, p)
	}
	sort.Strings(perms)
	return perms, nil
}

func (m *auth) availableAccountPermissions() (map[int64]string, error) {
	rows, err := m.db.Query(`SELECT id, name FROM account_available_permission`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	perms := make(map[int64]string)
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		perms[id] = name
	}
	return perms, rows.Err()
}

func (m *auth) groupNames() (map[int64]string, error) {
	rows, err := m.db.Query(`SELECT id, name FROM account_group`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groupNames := make(map[int64]string)
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		groupNames[id] = name
	}
	return groupNames, rows.Err()
}

func (m *auth) AvailableAccountGroups(withPermissions bool) ([]*common.AccountGroup, error) {
	if withPermissions {
		groupNames, err := m.groupNames()
		if err != nil {
			return nil, err
		}

		rows, err := m.db.Query(`SELECT group_id, permission_id FROM account_group_permission ORDER BY group_id`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var groups []*common.AccountGroup
		var group *common.AccountGroup
		for rows.Next() {
			var groupID, permID int64
			if err := rows.Scan(&groupID, &permID); err != nil {
				return nil, err
			}

			if group == nil || groupID != group.ID {
				// TODO: There's a small timing issue if a group is created between the
				// query to get the group names and the query to fetch the permissions.
				// It should be rare and mostly harmless so ignoring it for now.
				group = &common.AccountGroup{
					ID:   groupID,
					Name: groupNames[groupID],
				}
				// Track which groups have been seen so we can include groups without
				// any permissions later.
				delete(groupNames, groupID)
				groups = append(groups, group)
			}

			group.Permissions = append(group.Permissions, m.perms[permID])
		}

		// Include groups that don't have any permissions attached
		for id, name := range groupNames {
			groups = append(groups, &common.AccountGroup{ID: id, Name: name})
		}

		sort.Sort(byAccountGroupName(groups))

		return groups, rows.Err()
	}

	rows, err := m.db.Query(`SELECT id, name FROM account_group	ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*common.AccountGroup
	for rows.Next() {
		group := &common.AccountGroup{}
		if err := rows.Scan(&group.ID, &group.Name); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

func (m *auth) PermissionsForAccount(accountID int64) ([]string, error) {
	rows, err := m.db.Query(`
		SELECT DISTINCT permission_id
			FROM account_group_member
			INNER JOIN account_group_permission ON account_group_permission.group_id = account_group_member.group_id
			WHERE account_group_member.account_id = ?
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if p := m.perms[id]; p == "" {
			golog.Errorf("Unknown account permission ID %d. Cache out of date", id)
		} else {
			perms = append(perms, p)
		}
	}

	return perms, rows.Err()
}

func (m *auth) GroupsForAccount(accountID int64) ([]*common.AccountGroup, error) {
	rows, err := m.db.Query(`
		SELECT account_group_member.group_id, account_group.name
		FROM account_group_member
		INNER JOIN account_group ON account_group.id = account_group_member.group_id
		WHERE account_group_member.account_id = ?`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*common.AccountGroup
	for rows.Next() {
		group := &common.AccountGroup{}
		if err := rows.Scan(&group.ID, &group.Name); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

func (m *auth) UpdateGroupsForAccount(accountID int64, groups map[int64]bool) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}

	for groupID, state := range groups {
		var err error
		if state {
			_, err = tx.Exec(`REPLACE INTO account_group_member (account_id, group_id) VALUES (?, ?)`, accountID, groupID)
		} else {
			_, err = tx.Exec(`DELETE FROM account_group_member WHERE account_id = ? AND group_id = ?`, accountID, groupID)
		}
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
