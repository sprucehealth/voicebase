package test_api

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestAccountPermissions(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	perms, err := testData.AuthAPI.AvailableAccountPermissions()
	test.OK(t, err)
	test.Assert(t, len(perms) != 0, "Permissions list of empty")
	for _, p := range perms {
		test.Assert(t, p != "", "Empty permission string")
	}

	groups, err := testData.AuthAPI.AvailableAccountGroups(true)
	test.OK(t, err)
	test.Assert(t, len(groups) != 0, "Groups list is empty")
	for _, g := range groups {
		test.Assert(t, len(g.Permissions) != 0, "Group %s has empty permissions list", g.Name)
		test.Assert(t, g.Name != "", "Group with no name")
		test.Assert(t, g.ID != 0, "Group with no ID")
	}

	accountID, err := testData.AuthAPI.CreateAccount("test+perms@sprucehealth.com", "xyz", api.RoleAdmin)
	test.OK(t, err)

	actGroups, err := testData.AuthAPI.GroupsForAccount(accountID)
	test.OK(t, err)
	test.Equals(t, 0, len(actGroups))

	perms, err = testData.AuthAPI.PermissionsForAccount(accountID)
	test.OK(t, err)
	test.Equals(t, 0, len(perms))

	test.OK(t, testData.AuthAPI.UpdateGroupsForAccount(accountID, map[int64]bool{
		groups[0].ID: true,
		groups[1].ID: true,
		groups[2].ID: false,
	}))

	actGroups, err = testData.AuthAPI.GroupsForAccount(accountID)
	test.OK(t, err)
	test.Equals(t, 2, len(actGroups))

	perms, err = testData.AuthAPI.PermissionsForAccount(accountID)
	test.OK(t, err)
	test.Equals(t, countUnique(append(groups[0].Permissions, groups[1].Permissions...)), len(perms))

	test.OK(t, testData.AuthAPI.UpdateGroupsForAccount(accountID, map[int64]bool{
		groups[1].ID: false,
	}))

	perms, err = testData.AuthAPI.PermissionsForAccount(accountID)
	test.OK(t, err)
	test.Equals(t, len(groups[0].Permissions), len(perms))
}

func countUnique(ss []string) int {
	var count int
	seen := make(map[string]struct{})
	for _, s := range ss {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			count++
		}
	}
	return count
}
