package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/patientcase"
	"github.com/sprucehealth/backend/common"
)

type careTeamCmd struct {
	dataAPI api.DataAPI
}

func newCareTeamCmd(dataAPI api.DataAPI, pcSvc patientcase.Service) (command, error) {
	return &careTeamCmd{
		dataAPI: dataAPI,
	}, nil
}

func (c *careTeamCmd) run(args []string) error {
	fs := flag.NewFlagSet("careteam", flag.ExitOnError)
	patientID := fs.Uint64("patient_id", 0, "`ID` of the patient who's case it is")
	if err := fs.Parse(args); err != nil {
		return err
	}

	scn := bufio.NewScanner(os.Stdin)

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
	fmt.Printf("Patient: %s %s\n", patient.FirstName, patient.LastName)

	careTeam, err := c.dataAPI.GetActiveMembersOfCareTeamForPatient(pid, true)
	if err != nil {
		return fmt.Errorf("Failed to get care team: %s", err)
	}
	w := tabwriter.NewWriter(os.Stdout, 4, 8, 4, ' ', 0)
	fmt.Println("Active care team:")
	fmt.Fprintf(w, "\tID\tLong Display Name\tStatus\n")
	for _, ct := range careTeam {
		fmt.Fprintf(w, "\t%d\t%s\t%s\n", ct.ProviderID, ct.LongDisplayName, ct.Status)
	}
	return w.Flush()
}
