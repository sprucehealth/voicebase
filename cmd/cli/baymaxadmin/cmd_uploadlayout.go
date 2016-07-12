package main

import (
	"archive/zip"
	"bufio"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"context"

	"github.com/sajari/docconv"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/layout"
)

const (
	sharedSAMLLogicFilename = "Shared SAML logic.docx"
)

// uploadLayoutCmd ingests the SAML content into the system and replaced the current category set available to users.
// The content directory is a zipped download of the folder located at the following google drive location:
// https://drive.google.com/open?id=0BzrunShFLnqYMW1fel9iQ1dUcmc
// The script uses the category set file as the directive for what the categories should look like. Any existing visit layouts are
// updated, and if they don't exist new ones are created. Any previously existing layouts that don't match the names of any
// visit layouts in the category set are marked as deleted.
type uploadLayoutCmd struct {
	cnf         *config
	layoutCli   layout.LayoutClient
	layoutStore layout.Storage
}

func newUploadLayoutCmd(cnf *config) (command, error) {
	layoutCli, err := cnf.layoutClient()
	if err != nil {
		return nil, err
	}

	return &uploadLayoutCmd{
		cnf:       cnf,
		layoutCli: layoutCli,
	}, nil
}

func (c *uploadLayoutCmd) run(args []string) error {
	fs := flag.NewFlagSet("uploadlayout", flag.ExitOnError)
	zippedSAMLFolder := fs.String("zipped_content_directory", "", "zipped folder downloaded from google drive that contains all SAML")
	categorySetFileName := fs.String("category_set_filename", "", "filename of the file containing breakdown of categories and the corresponding visits to include")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *zippedSAMLFolder == "" {
		return errors.Trace(fmt.Errorf("zipped_content_directory required"))
	}
	if *categorySetFileName == "" {
		return errors.Trace(fmt.Errorf("category_set_filename required"))
	}

	r, err := zip.OpenReader(*zippedSAMLFolder)
	if err != nil {
		return errors.Trace(err)
	}
	defer r.Close()

	categoryInfos, err := parseVisitsBreakdown(*categorySetFileName)
	if err != nil {
		return errors.Trace(err)
	}

	fileMap := make(map[string]*zip.File, len(r.File))
	var sharedSAMLLogicFile *zip.File
	for _, f := range r.File {
		_, fn := path.Split(f.Name)

		if fn == sharedSAMLLogicFilename {
			sharedSAMLLogicFile = f
		}

		if samlTitle, ok := isSAMLFile(fn); ok {
			fileMap[samlTitle] = f
		}

		fileMap[fn] = f
	}

	// ensure that all visits to be created according to breakdown are in filemap
	filesNotFound := make([]string, 0)
	for _, categoryInfo := range categoryInfos {
		for _, visitInfo := range categoryInfo.visitInfos {
			if _, ok := fileMap[visitInfo.name]; !ok {
				filesNotFound = append(filesNotFound, visitInfo.name)
			}
		}
	}
	if len(filesNotFound) > 0 {
		return errors.Trace(fmt.Errorf("The following files do not exist in the content directory: %v", filesNotFound))
	}

	sharedSAMLLogicReader, err := sharedSAMLLogicFile.Open()
	if err != nil {
		return errors.Trace(err)
	}
	defer sharedSAMLLogicReader.Close()

	sharedSAMLLogicData, _, err := docconv.ConvertDocx(sharedSAMLLogicReader)
	if err != nil {
		return errors.Trace(err)
	}

	// get the existing visit categories and the correspdoning visits
	existingCategories, visitMap, err := c.existingCategoriesAndVisitLayouts()
	if err != nil {
		return errors.Trace(err)
	}

	// rename or create new categories to match the categories
	// that should exist
	categoriesMarked := make(map[string]struct{})
	categoriesCreated := make([]*layout.VisitCategory, 0, len(categoryInfos))
	categoriesUpdated := make([]*layout.VisitCategory, 0, len(categoryInfos))
	for i, categoryInfo := range categoryInfos {
		if i >= len(existingCategories) {
			createCategoryRes, err := c.layoutCli.CreateVisitCategory(context.Background(), &layout.CreateVisitCategoryRequest{
				Name: categoryInfo.name,
			})
			if err != nil {
				return errors.Trace(fmt.Errorf("Unable to create category %s: %s", categoryInfo.name, err))
			}
			categoryInfo.categoryInSystem = createCategoryRes.Category
			categoriesCreated = append(categoriesCreated, createCategoryRes.Category)
		} else {
			updateCategoryRes, err := c.layoutCli.UpdateVisitCategory(context.Background(), &layout.UpdateVisitCategoryRequest{
				VisitCategoryID: existingCategories[i].ID,
				Name:            categoryInfo.name,
			})
			if err != nil {
				return errors.Trace(fmt.Errorf("Unable to update cateogry name for %s: %s", existingCategories[i].ID, err))
			}
			categoriesMarked[existingCategories[i].ID] = struct{}{}
			categoryInfo.categoryInSystem = updateCategoryRes.Category
			categoriesUpdated = append(categoriesUpdated, updateCategoryRes.Category)
		}
	}

	// within each of the categories, either update or create a visit
	visitsMarked := make(map[string]struct{})
	visitLayoutsCreated := make([]*layout.VisitLayout, 0, len(visitMap))
	visitLayoutsUpdated := make([]*layout.VisitLayout, 0, len(visitMap))
	for _, categoryInfo := range categoryInfos {
		for _, visitInfo := range categoryInfo.visitInfos {
			// update the existing visit if one already exists by its name
			if existingVisitLayout, ok := visitMap[visitInfo.name]; ok {

				// update
				saml, err := populateSAMLToUpload(sharedSAMLLogicData, fileMap[visitInfo.name])
				if err != nil {
					return errors.Trace(err)
				}

				_, err = c.layoutCli.UpdateVisitLayout(context.Background(), &layout.UpdateVisitLayoutRequest{
					SAML:           saml,
					InternalName:   parseVisitName(visitInfo.name),
					UpdateSAML:     true,
					VisitLayoutID:  existingVisitLayout.ID,
					UpdateCategory: true,
					CategoryID:     categoryInfo.categoryInSystem.ID,
				})
				visitsMarked[existingVisitLayout.ID] = struct{}{}
				if err != nil {
					golog.Errorf("Unable to update saml for '%s'. Skipping... %s", existingVisitLayout.Name, err)
					continue
				}

				visitsMarked[existingVisitLayout.ID] = struct{}{}
				visitLayoutsUpdated = append(visitLayoutsUpdated, existingVisitLayout)
			} else {
				// create
				saml, err := populateSAMLToUpload(sharedSAMLLogicData, fileMap[visitInfo.name])
				if err != nil {
					return errors.Trace(fmt.Errorf("unable to create visit layout for '%s': %s", visitInfo.name, err))
				}

				createVisitLayoutRes, err := c.layoutCli.CreateVisitLayout(context.Background(), &layout.CreateVisitLayoutRequest{
					CategoryID:   categoryInfo.categoryInSystem.ID,
					Name:         parseVisitName(visitInfo.name),
					InternalName: visitInfo.name,
					SAML:         saml,
				})
				if err != nil {
					golog.Errorf("Unable to create visit layout for '%s'. Skipping...: %s", visitInfo.name, err)
					continue
				}
				visitLayoutsCreated = append(visitLayoutsCreated, createVisitLayoutRes.VisitLayout)
			}
		}
	}

	// for the pre-existing categories or visits not found in the category set, mark them to be deleted
	visitLayoutsDeleted := make([]*layout.VisitLayout, 0, len(visitMap))
	for _, visitLayout := range visitMap {
		if _, ok := visitsMarked[visitLayout.ID]; !ok {
			_, err := c.layoutCli.DeleteVisitLayout(context.Background(), &layout.DeleteVisitLayoutRequest{
				VisitLayoutID: visitLayout.ID,
			})
			if err != nil {
				return errors.Trace(fmt.Errorf("unable to delete visit layout %s : %s", visitLayout.ID, err))
			}
			visitLayoutsDeleted = append(visitLayoutsDeleted, visitLayout)
		}
	}

	categoriesDeleted := make([]*layout.VisitCategory, 0, len(existingCategories))
	for _, category := range existingCategories {
		if _, ok := categoriesMarked[category.ID]; !ok {
			_, err := c.layoutCli.DeleteVisitCategory(context.Background(), &layout.DeleteVisitCategoryRequest{
				VisitCategoryID: category.ID,
			})
			if err != nil {
				return errors.Trace(fmt.Errorf("unable to delete visit category %s : %s", category.ID, err))
			}
			categoriesDeleted = append(categoriesDeleted, category)
		}
	}

	fmt.Println("Successfully updated category set!")

	fmt.Println("Categories created:")
	printCategories(categoriesCreated)
	fmt.Println("Categories updated:")
	printCategories(categoriesUpdated)
	fmt.Println("Categories deleted:")
	printCategories(categoriesDeleted)

	fmt.Println("VisitLayouts created:")
	printVisitLayouts(visitLayoutsCreated)
	fmt.Println("VisitLayouts updated:")
	printVisitLayouts(visitLayoutsUpdated)
	fmt.Println("VisitLayouts deleted:")
	printVisitLayouts(visitLayoutsDeleted)

	return nil
}

func isSAMLFile(fn string) (string, bool) {
	// only consider a file as an algorithm if its name starts with Spruce
	if !strings.HasPrefix(fn, "Spruce ") {
		return "", false
	}

	// don't consider spruce guide a SAML file
	if strings.HasPrefix(fn, "Spruce Guide_") {
		return "", false
	}

	if !strings.Contains(fn, "Algorithm") {
		return "", false
	}

	replacer := strings.NewReplacer("Spruce ", "", " Algorithm", "", ".docx", "")

	return replacer.Replace(fn), true
}

func parseVisitName(s string) string {
	replacer := strings.NewReplacer(" (brief)", "")
	return replacer.Replace(s)
}

type categoryInfo struct {
	name             string
	categoryInSystem *layout.VisitCategory
	visitInfos       []*visitInfo
}

type visitInfo struct {
	name string
}

// parseVisitsBreakdown processes the file at the specified location
// to determine the categories and visits.
func parseVisitsBreakdown(fileName string) ([]*categoryInfo, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, errors.Trace(err)
	}
	scanner := bufio.NewScanner(f)

	var categories []*categoryInfo
	for scanner.Scan() {
		currentCategory := &categoryInfo{
			name: scanner.Text(),
		}
		for scanner.Scan() {
			if scanner.Text() == "" {
				// end of category
				break
			}
			currentCategory.visitInfos = append(currentCategory.visitInfos, &visitInfo{
				name: scanner.Text(),
			})
		}
		categories = append(categories, currentCategory)
	}

	return categories, nil
}

func populateSAMLToUpload(sharedSAMLData string, file *zip.File) (string, error) {
	fileReader, err := file.Open()
	if err != nil {
		return "", errors.Trace(err)
	}
	defer fileReader.Close()

	fileData, _, err := docconv.ConvertDocx(fileReader)
	if err != nil {
		return "", errors.Trace(err)
	}

	return sharedSAMLData + "\n" + fileData, nil
}

func (c *uploadLayoutCmd) existingCategoriesAndVisitLayouts() ([]*layout.VisitCategory, map[string]*layout.VisitLayout, error) {
	visitCategoriesRes, err := c.layoutCli.ListVisitCategories(context.Background(), &layout.ListVisitCategoriesRequest{})
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	existingCategories := visitCategoriesRes.Categories
	visitMap := make(map[string]*layout.VisitLayout, len(existingCategories))
	for _, category := range existingCategories {
		visitLayoutsRes, err := c.layoutCli.ListVisitLayouts(context.Background(), &layout.ListVisitLayoutsRequest{
			VisitCategoryID: category.ID,
		})
		if err != nil {
			return nil, nil, errors.Trace(err)
		}

		for _, visitLayout := range visitLayoutsRes.VisitLayouts {
			visitMap[visitLayout.InternalName] = visitLayout
		}
	}

	return existingCategories, visitMap, nil
}

func printVisitLayouts(visitLayouts []*layout.VisitLayout) {
	if len(visitLayouts) == 0 {
		fmt.Println("NONE")
	}
	for _, visitLayout := range visitLayouts {
		fmt.Printf("ID:%s\tName: %s\n", visitLayout.ID, visitLayout.Name)
	}
}

func printCategories(categories []*layout.VisitCategory) {
	if len(categories) == 0 {
		fmt.Println("NONE")
	}
	for _, visitCategory := range categories {
		fmt.Printf("ID:%s\tName: %s\n", visitCategory.ID, visitCategory.Name)
	}
}
