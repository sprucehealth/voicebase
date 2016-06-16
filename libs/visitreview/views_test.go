package visitreview

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/mapstructure"
)

func TestParsingMediaListView(t *testing.T) {

	review := `{
    "type": "d_visit_review:standard_media_section",
    "sections" : [{
      "type": "d_visit_review:standard_media_subsection",
      "view": {
        "type": "d_visit_review:title_media_items_list",
        "content_config": {
          "key": "test:media"
        }
      }
    }]
  }`

	var jsonMap map[string]interface{}
	if err := json.Unmarshal([]byte(review), &jsonMap); err != nil {
		t.Fatal(err)
	}

	var sectionList SectionListView
	decoderConfig := &mapstructure.DecoderConfig{
		Result:   &sectionList,
		TagName:  "json",
		Registry: *TypeRegistry,
	}

	d, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		t.Fatal(err)
	}

	if err := d.Decode(jsonMap); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, SectionListView{
		Type: "d_visit_review:standard_media_section",
		Sections: []View{
			&StandardMediaSubsectionView{
				Type: "d_visit_review:standard_media_subsection",
				SubsectionView: &TitleMediaItemsListView{
					Type: "d_visit_review:title_media_items_list",
					ContentConfig: &ContentConfig{
						Key: "test:media",
					},
				},
			},
		},
	}, sectionList)

	// render

	viewContext := NewViewContext(map[string]interface{}{
		"test:media": []TitleMediaListData{
			{
				Title: "test",
				Media: []MediaData{
					{
						Title:   "media1",
						MediaID: "mediaID1",
						URL:     "http://test.com/mediaID1",
						Type:    "photo",
					},
				},
			},
		},
	})

	renderedView, err := sectionList.Render(viewContext)
	test.OK(t, err)

	jsonData, err := json.Marshal(renderedView)
	test.OK(t, err)
	test.Equals(t, "{\"sections\":[{\"type\":\"d_visit_review:standard_media_subsection\",\"view\":{\"items\":[{\"title\":\"test\",\"media\":[{\"title\":\"media1\",\"media_id\":\"mediaID1\",\"url\":\"http://test.com/mediaID1\",\"type\":\"photo\",\"placeholder_url\":\"\"}]}],\"type\":\"d_visit_review:title_media_items_list\"}}],\"type\":\"d_visit_review:sections_list\"}", string(jsonData))
}
