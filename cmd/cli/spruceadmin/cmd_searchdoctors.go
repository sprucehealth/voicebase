package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/patientcase"
	"github.com/sprucehealth/backend/libs/errors"
)

type searchDoctorsCmd struct {
	dataAPI api.DataAPI
}

func newSearchDoctorsCmd(dataAPI api.DataAPI, pcSvc patientcase.Service) (command, error) {
	return &searchDoctorsCmd{
		dataAPI: dataAPI,
	}, nil
}

func (c *searchDoctorsCmd) run(args []string) error {
	if len(args) == 0 {
		return errors.New("query required")
	}
	query := args[0]

	drs, err := c.dataAPI.SearchDoctors(query)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 4, 8, 4, ' ', 0)
	fmt.Fprintf(w, "ID\tAccount ID\tFirst Name\tLast Name\tEmail\n")
	for _, dr := range drs {
		fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%s\n", dr.DoctorID, dr.AccountID, dr.FirstName, dr.LastName, dr.Email)
	}
	return w.Flush()
}
