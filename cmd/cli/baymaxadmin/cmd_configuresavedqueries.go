package main

import (
	"bufio"
	"context"

	"flag"
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

type configureSavedQueriesCmd struct {
	cnf          *config
	dirCli       directory.DirectoryClient
	threadingCli threading.ThreadsClient
}

func newConfigureSavedQueriesCmd(cnf *config) (command, error) {
	dirCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}

	threadingCli, err := cnf.threadingClient()
	if err != nil {
		return nil, err
	}

	return &configureSavedQueriesCmd{
		cnf:          cnf,
		dirCli:       dirCli,
		threadingCli: threadingCli,
	}, nil
}

type savedQuery struct {
	ShortTitle           string
	LongTitle            string
	Description          string
	NotificationsEnabled bool
	Ordinal              int
	Query                string
	Template             bool
	Hidden               bool
	Type                 string
}

type savedQueriesConfig struct {
	SavedQueries []*savedQuery
}

func (c *configureSavedQueriesCmd) run(args []string) error {
	fs := flag.NewFlagSet("configuresavedqueries", flag.ExitOnError)
	entityID := fs.String("entity_id", "", "ID of the entity for which to create saved queries")
	configFile := fs.String("config_toml_file", "", "File containing config in toml form")
	force := fs.Bool("force", false, "force first deletes the existing saved queries to then create new ones")

	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *entityID == "" {
		*entityID = prompt(scn, "Entity ID:")
	}
	if *entityID == "" {
		return errors.Errorf("Entity ID required")
	}

	if *configFile == "" {
		*configFile = prompt(scn, "Config file name:")
	}
	if *configFile == "" {
		return errors.Errorf("Config file required")
	}

	fileData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return errors.Trace(err)
	}

	var sqc savedQueriesConfig
	if _, err := toml.Decode(string(fileData), &sqc); err != nil {
		return errors.Trace(err)
	}

	ctx := context.Background()
	ent, err := directory.SingleEntity(ctx, c.dirCli, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: *entityID,
		},
		RootTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		return errors.Trace(err)
	}

	var savedQueryIDs []string
	var savedQueries []*threading.SavedQuery
	if ent.Type == directory.EntityType_ORGANIZATION {
		savedQueriesRes, err := c.threadingCli.SavedQueryTemplates(ctx, &threading.SavedQueryTemplatesRequest{
			EntityID: ent.ID,
		})
		if err != nil {
			return errors.Trace(err)
		}
		for _, sq := range savedQueriesRes.SavedQueries {
			savedQueryIDs = append(savedQueryIDs, sq.ID)
		}
		savedQueries = savedQueriesRes.SavedQueries
	} else {
		savedQueriesRes, err := c.threadingCli.SavedQueries(ctx, &threading.SavedQueriesRequest{
			EntityID: ent.ID,
		})
		if err != nil {
			return errors.Trace(err)
		}
		for _, sq := range savedQueriesRes.SavedQueries {
			savedQueryIDs = append(savedQueryIDs, sq.ID)
		}
		savedQueries = savedQueriesRes.SavedQueries
	}

	existingSavedQueries := make(map[string]struct{})
	if *force {
		// delete existing saved thread queries
		if _, err := c.threadingCli.DeleteSavedQueries(ctx, &threading.DeleteSavedQueriesRequest{
			SavedQueryIDs: savedQueryIDs,
		}); err != nil {
			return errors.Trace(err)
		}
	} else {
		// get a map of existing saved queries by title
		for _, existingSQ := range savedQueries {
			existingSavedQueries[existingSQ.ShortTitle] = struct{}{}
		}
	}

	for _, savedQuery := range sqc.SavedQueries {

		// don't create if it already exists
		if _, ok := existingSavedQueries[savedQuery.ShortTitle]; ok {
			golog.Warningf("Skipping creation saved query with title %s since it already exists", savedQuery.ShortTitle)
			continue
		}

		query, err := threading.ParseQuery(savedQuery.Query)
		if err != nil {
			return errors.Errorf("Unable to parse query '%s' : %s", savedQuery.Query, err)
		}

		savedQueryType := threading.SavedQueryType_value[savedQuery.Type]
		if threading.SavedQueryType(savedQueryType) == threading.SAVED_QUERY_TYPE_INVALID {
			return errors.Errorf("Invalid saved query type: %s", savedQuery.Type)
		}

		if _, err := c.threadingCli.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
			EntityID:             *entityID,
			ShortTitle:           savedQuery.ShortTitle,
			LongTitle:            savedQuery.LongTitle,
			Description:          savedQuery.Description,
			Template:             savedQuery.Template,
			Ordinal:              int32(savedQuery.Ordinal),
			NotificationsEnabled: savedQuery.NotificationsEnabled,
			Query:                query,
			Hidden:               savedQuery.Hidden,
			Type:                 threading.SavedQueryType(savedQueryType),
		}); err != nil {
			return errors.Errorf("Unable to create saved query %s : %s", savedQuery.ShortTitle, err)
		}

		golog.Infof("Created saved query %+v", savedQuery)
	}
	return nil
}
