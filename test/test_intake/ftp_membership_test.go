package test_intake

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestFTPMembershipsNone(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctor := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	memberships, err := testData.DataAPI.FTPMembershipsForDoctor(doctor.DoctorID)
	test.OK(t, err)
	test.Equals(t, 0, len(memberships))
}

func TestFTPMembershipCreation(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pID := initializePathway(t, testData, "pathway")
	doctor := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor2 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	ftpID, err := testData.DataAPI.InsertFavoriteTreatmentPlan(createFTP("My FTP", doctor2.DoctorID), "pathway", 0)
	test.OK(t, err)
	memberships, err := testData.DataAPI.FTPMembershipsForDoctor(doctor.DoctorID)
	test.OK(t, err)
	test.Equals(t, 0, len(memberships))
	memberships, err = testData.DataAPI.FTPMembershipsForDoctor(doctor2.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(memberships))
	_, err = testData.DataAPI.CreateFTPMembership(ftpID, doctor.DoctorID, pID)
	test.OK(t, err)
	memberships, err = testData.DataAPI.FTPMembershipsForDoctor(doctor.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(memberships))
}

func TestFTPMembershipDeletion(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pID := initializePathway(t, testData, "pathway")
	doctor := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	ftpID, err := testData.DataAPI.InsertFavoriteTreatmentPlan(createFTP("My FTP", doctor.DoctorID), "pathway", 0)
	test.OK(t, err)
	memberships, err := testData.DataAPI.FTPMembershipsForDoctor(doctor.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(memberships))
	_, err = testData.DataAPI.DeleteFTPMembership(ftpID, doctor.DoctorID, pID)
	test.OK(t, err)
	memberships, err = testData.DataAPI.FTPMembershipsForDoctor(doctor.DoctorID)
	test.OK(t, err)
	test.Equals(t, 0, len(memberships))
}

func TestFTPMembershipMultiplePathwayMemberships(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	initializePathway(t, testData, "pathway")
	pID2 := initializePathway(t, testData, "pathway2")
	doctor := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	ftpID, err := testData.DataAPI.InsertFavoriteTreatmentPlan(createFTP("My FTP", doctor.DoctorID), "pathway", 0)
	test.OK(t, err)
	memberships, err := testData.DataAPI.FTPMembershipsForDoctor(doctor.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(memberships))
	_, err = testData.DataAPI.CreateFTPMembership(ftpID, doctor.DoctorID, pID2)
	test.OK(t, err)
	memberships, err = testData.DataAPI.FTPMembershipsForDoctor(doctor.DoctorID)
	test.OK(t, err)
	test.Equals(t, 2, len(memberships))
}

func TestFTPMembershipQuery(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	pID := initializePathway(t, testData, "pathway")
	pID2 := initializePathway(t, testData, "pathway2")
	doctor := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	doctor2 := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	ftpID, err := testData.DataAPI.InsertFavoriteTreatmentPlan(createFTP("My FTP", doctor2.DoctorID), "pathway", 0)
	test.OK(t, err)
	memberships, err := testData.DataAPI.FTPMembershipsForDoctor(doctor.DoctorID)
	test.OK(t, err)
	test.Equals(t, 0, len(memberships))
	memberships, err = testData.DataAPI.FTPMembershipsForDoctor(doctor2.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(memberships))
	_, err = testData.DataAPI.CreateFTPMembership(ftpID, doctor.DoctorID, pID)
	test.OK(t, err)
	memberships, err = testData.DataAPI.FTPMembershipsForDoctor(doctor.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(memberships))
	_, err = testData.DataAPI.CreateFTPMembership(ftpID, doctor.DoctorID, pID2)
	test.OK(t, err)
	memberships, err = testData.DataAPI.FTPMemberships(ftpID)
	test.OK(t, err)
	test.Equals(t, 3, len(memberships))
}

func initializePathway(t *testing.T, testData *test_integration.TestData, name string) int64 {
	testData.DataAPI.CreatePathway(&common.Pathway{
		Tag:            name,
		Name:           name,
		MedicineBranch: name,
		Status:         common.PathwayActive,
		Details:        nil,
	})
	pathway, err := testData.DataAPI.PathwayForTag(name, api.PONone)
	test.OK(t, err)
	return pathway.ID
}

func createFTP(name string, creatorID int64) *common.FavoriteTreatmentPlan {
	ftp := &common.FavoriteTreatmentPlan{
		Name:              name,
		CreatorID:         &creatorID,
		ParentID:          nil,
		RegimenPlan:       nil,
		TreatmentList:     nil,
		Note:              "My Note",
		ScheduledMessages: nil,
		ResourceGuides:    nil,
	}
	return ftp
}
