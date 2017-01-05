package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/patientcase"
)

type moveCaseCmd struct {
	dataAPI api.DataAPI
	pcSvc   patientcase.Service
}

func newMoveCaseCmd(cnf *conf) (command, error) {
	return &moveCaseCmd{
		dataAPI: cnf.DataAPI,
		pcSvc:   cnf.patientCaseService(),
	}, nil
}

func (c *moveCaseCmd) run(args []string) error {
	fs := flag.NewFlagSet("movecase", flag.ExitOnError)
	doctorID := fs.Uint64("new_doctor_id", 0, "`ID` of doctor to receive the case")
	ccID := fs.Uint64("cc_id", 0, "`ID` of the care coordinator who is moving the case")
	patientID := fs.Uint64("patient_id", 0, "`ID` of the patient who's case it is")
	caseID := fs.Uint64("case_id", 0, "`ID` of the case to migrate (if not provided, a list of cases for the patient will be displayed)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	scn := bufio.NewScanner(os.Stdin)

	// new doctor
	if *doctorID == 0 {
		var err error
		*doctorID, err = strconv.ParseUint(prompt(scn, "Doctor ID: "), 10, 64)
		if err != nil {
			return fmt.Errorf("Invalid ID")
		}
	}
	newDoctor, err := c.dataAPI.Doctor(int64(*doctorID), true)
	if err != nil {
		return fmt.Errorf("Could not get new doctor: %s", err)
	}
	fmt.Printf("Migrating to doctor: %s\n", newDoctor.LongDisplayName)

	// care coordinator
	if *ccID == 0 {
		var err error
		*ccID, err = strconv.ParseUint(prompt(scn, "CC ID: "), 10, 64)
		if err != nil {
			return fmt.Errorf("Invalid ID")
		}
	}
	cc, err := c.dataAPI.Doctor(int64(*ccID), true)
	if err != nil {
		return fmt.Errorf("Failed to get CC: %s", err)
	}
	fmt.Printf("CC: %s\n", cc.LongDisplayName)

	// patient
	if *patientID == 0 {
		var err error
		*patientID, err = strconv.ParseUint(prompt(scn, "Patient ID: "), 10, 64)
		if err != nil {
			return fmt.Errorf("Invalid ID")
		}
	}
	pid := common.NewPatientID(*patientID)
	patient, err := c.dataAPI.Patient(pid, true)
	if err != nil {
		return fmt.Errorf("Failed to get patient: %s", err)
	}
	patientInitials := patient.FirstName[:1]
	if len(patient.LastName) > 0 {
		patientInitials += patient.LastName[:1]
	}
	fmt.Printf("Patient Initials: %s\n", patientInitials)

	if *caseID == 0 {
		cases, err := c.dataAPI.GetCasesForPatient(patient.ID, nil)
		if err != nil {
			return fmt.Errorf("Failed to get list of cases for patient: %s", err)
		}
		if len(cases) == 0 {
			return errors.New("No cases to migrate")
		}
		fmt.Printf("Cases:\n")
		w := tabwriter.NewWriter(os.Stdout, 4, 8, 4, ' ', 0)
		fmt.Fprintf(w, "\tID\tName\tStatus\n")
		for _, c := range cases {
			fmt.Fprintf(w, "\t%s\t%s\t%s\n", c.ID, c.Name, c.Status)
		}
		if err := w.Flush(); err != nil {
			return err
		}
		*caseID, err = strconv.ParseUint(prompt(scn, "Case ID: "), 10, 64)
		if err != nil {
			return errors.New("Invalid case ID")
		}
	}

	pcase, err := c.dataAPI.GetPatientCaseFromID(int64(*caseID))
	if err != nil {
		return fmt.Errorf("Failed to get the case: %s", err)
	}
	fmt.Printf("Case %s: %s (status %s)\n", pcase.ID, pcase.Name, pcase.Status)
	if pcase.PatientID.Uint64() != patient.ID.Uint64() {
		return fmt.Errorf("Patient ID of case %s does not match expected ID %s", pcase.PatientID, patient.ID)
	}

	visits, err := c.dataAPI.GetVisitsForCase(pcase.ID.Int64(), nil)
	if err != nil {
		return fmt.Errorf("Failed to get visits: %s", err)
	}
	fmt.Printf("Visits:\n")
	w := tabwriter.NewWriter(os.Stdout, 4, 8, 4, ' ', 0)
	fmt.Fprintf(w, "\tID\tPathway Tag\tSKU Type\tStatus\n")
	for _, v := range visits {
		fmt.Fprintf(w, "\t%s\t%s\t%s\t%s\n", v.ID, v.PathwayTag, v.SKUType, v.Status)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	fmt.Printf("Proceed [y/N]? ")
	if !scn.Scan() {
		return nil
	}
	if strings.ToLower(scn.Text()) != "y" {
		return nil
	}

	return c.pcSvc.ChangeCareProvider(pcase.ID.Int64(), newDoctor.ID.Int64(), cc.ID.Int64())
}
