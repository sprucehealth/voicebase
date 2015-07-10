package test

import (
	"github.com/sprucehealth/backend/tagging"
	"github.com/sprucehealth/backend/tagging/model"
	"github.com/sprucehealth/backend/tagging/response"
)

type TestTaggingClient struct {
	TagsCreated map[int64][]string
	TagsDeleted map[int64][]string
}

func (t *TestTaggingClient) CaseAssociations(ms []*model.TagMembership, start, end int64) ([]*response.TagAssociation, error) {
	return nil, nil
}
func (t *TestTaggingClient) CaseTagMemberships(caseID int64) (map[string]*model.TagMembership, error) {
	return nil, nil
}
func (t *TestTaggingClient) DeleteTag(id int64) (int64, error) {
	return 0, nil
}
func (t *TestTaggingClient) DeleteTagCaseAssociation(text string, caseID int64) error {
	if t.TagsDeleted == nil {
		t.TagsDeleted = make(map[int64][]string)
	}
	t.TagsDeleted[caseID] = append(t.TagsDeleted[caseID], text)
	return nil
}
func (t *TestTaggingClient) DeleteTagCaseMembership(tagID, caseID int64) error {
	return nil
}
func (t *TestTaggingClient) InsertTagAssociation(tag *model.Tag, membership *model.TagMembership) (int64, error) {
	if t.TagsCreated == nil {
		t.TagsCreated = make(map[int64][]string)
	}
	t.TagsCreated[*membership.CaseID] = append(t.TagsCreated[*membership.CaseID], tag.Text)
	return 0, nil
}
func (t *TestTaggingClient) TagMembershipQuery(query string, ops tagging.TaggingOption) ([]*model.TagMembership, error) {
	return nil, nil
}
func (t *TestTaggingClient) InsertTagSavedSearch(ss *model.TagSavedSearch) (int64, error) {
	return 0, nil
}
func (t *TestTaggingClient) DeleteTagSavedSearch(ssID int64) (int64, error) {
	return 0, nil
}
func (t *TestTaggingClient) InsertTag(tag *model.Tag) (int64, error) {
	return 0, nil
}
func (t *TestTaggingClient) TagSavedSearchs() ([]*model.TagSavedSearch, error) {
	return nil, nil
}
func (t *TestTaggingClient) UpdateTag(tag *model.TagUpdate) error {
	return nil
}
func (t *TestTaggingClient) UpdateTagCaseMembership(membership *model.TagMembershipUpdate) error {
	return nil
}
func (t *TestTaggingClient) TagFromText(tagText string) (*response.Tag, error) {
	return nil, nil
}
func (t *TestTaggingClient) TagsFromText(tagText []string, ops tagging.TaggingOption) ([]*response.Tag, error) {
	return nil, nil
}
func (t *TestTaggingClient) TagsForCases(ids []int64, ops tagging.TaggingOption) (map[int64][]*response.Tag, error) {
	return nil, nil
}
func (t *TestTaggingClient) Tags(ids []int64) (map[int64]*response.Tag, error) {
	return make(map[int64]*response.Tag), nil
}
