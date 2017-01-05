package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
)

type casesCmd struct {
	dataAPI api.DataAPI
}

func newCasesCmd(cnf *conf) (command, error) {
	return &casesCmd{
		dataAPI: cnf.DataAPI,
	}, nil
}

func (c *casesCmd) run(args []string) error {
	fs := flag.NewFlagSet("cases", flag.ExitOnError)
	withCareTeam := fs.Bool("care_team", false, "Display care team for cases")
	patientID := fs.Uint64("patient_id", 0, "`ID` of the patient for which to lookup cases")
	withVisits := fs.Bool("visits", false, "Display visits for cases")
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

	cases, err := c.dataAPI.GetCasesForPatient(common.NewPatientID(*patientID), nil)
	if err != nil {
		return fmt.Errorf("Failed to get cases: %s", err)
	}
	for _, pcase := range cases {
		fmt.Printf("Case %s: %s (status %s)\n", pcase.ID, pcase.Name, pcase.Status)
		fmt.Printf("    Patient ID: %s\n", pcase.PatientID)
		fmt.Printf("    Created: %s\n", pcase.CreationDate)
		if pcase.ClosedDate != nil {
			fmt.Printf("\tClosed: %s\n", *pcase.ClosedDate)
		}

		if *withCareTeam {
			careTeam, err := c.dataAPI.GetActiveMembersOfCareTeamForCase(pcase.ID.Int64(), true)
			if err != nil {
				return fmt.Errorf("Failed to get care team: %s", err)
			}
			w := tabwriter.NewWriter(os.Stdout, 4, 8, 4, ' ', 0)
			fmt.Println("Active care team:")
			fmt.Fprintf(w, "\tID\tLong Display Name\tStatus\n")
			for _, ct := range careTeam {
				fmt.Fprintf(w, "\t%d\t%s\t%s\n", ct.ProviderID, ct.LongDisplayName, ct.Status)
			}
			if err := w.Flush(); err != nil {
				return fmt.Errorf("Failed to flush writer: %s", err)
			}
		}

		if *withVisits {
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
				return fmt.Errorf("Failed to flush writer: %s", err)
			}
		}
	}

	return nil
}
