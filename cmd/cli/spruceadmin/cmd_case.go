package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
)

type caseCmd struct {
	dataAPI api.DataAPI
}

func newCaseCmd(cnf *conf) (command, error) {
	return &caseCmd{
		dataAPI: cnf.DataAPI,
	}, nil
}

func (c *caseCmd) run(args []string) error {
	fs := flag.NewFlagSet("case", flag.ExitOnError)
	caseID := fs.Uint64("case_id", 0, "`ID` of the case to view")
	if err := fs.Parse(args); err != nil {
		return err
	}

	scn := bufio.NewScanner(os.Stdin)

	// patient
	if *caseID == 0 {
		var err error
		*caseID, err = strconv.ParseUint(prompt(scn, "Case ID: "), 10, 64)
		if err != nil {
			return fmt.Errorf("Invalid ID")
		}
	}

	pcase, err := c.dataAPI.GetPatientCaseFromID(int64(*caseID))
	if err != nil {
		return fmt.Errorf("Failed to get case: %s", err)
	}
	fmt.Printf("Case %s: %s (status %s)\n", pcase.ID, pcase.Name, pcase.Status)
	fmt.Printf("    Patient ID: %s\n", pcase.PatientID)
	fmt.Printf("    Created: %s\n", pcase.CreationDate)
	if pcase.ClosedDate != nil {
		fmt.Printf("\tClosed: %s\n", *pcase.ClosedDate)
	}

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
		return err
	}

	visits, err := c.dataAPI.GetVisitsForCase(pcase.ID.Int64(), nil)
	if err != nil {
		return fmt.Errorf("Failed to get visits: %s", err)
	}
	fmt.Printf("Visits:\n")
	w = tabwriter.NewWriter(os.Stdout, 4, 8, 4, ' ', 0)
	fmt.Fprintf(w, "\tID\tPathway Tag\tSKU Type\tStatus\n")
	for _, v := range visits {
		fmt.Fprintf(w, "\t%s\t%s\t%s\t%s\n", v.ID, v.PathwayTag, v.SKUType, v.Status)
	}
	return w.Flush()
}
