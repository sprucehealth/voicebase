package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type cityPageHandler struct {
	refTemplate       *template.Template
	webURL            string
	staticResourceURL string
}

type doctorItem struct {
	IsSpruceDoctor  bool
	Description     string
	LongDisplayName string
	ProfileImageURL string
	Experience      string
	Specialties     []string `json:",omitempty"`
	ProfileURL      string
}

type spruceScoreSection struct {
	Score       string
	Description string
	Bullets     []string
}

type descriptionItemsSection struct {
	Description string
	Items       []string
}

func NewCityPageHandler(templateLoader *www.TemplateLoader, webURL, staticResourceURL string) httputil.ContextHandler {
	return &cityPageHandler{
		refTemplate:       templateLoader.MustLoadTemplate("citypage.html", "base.html", nil),
		webURL:            webURL,
		staticResourceURL: staticResourceURL,
	}
}

func (c *cityPageHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	doctors := []doctorItem{
		{
			IsSpruceDoctor:  true,
			LongDisplayName: "Dr. Jason Fung",
			Description:     "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis vel ipsum quis diam tincidunt lobortis. Pellentesque egestas lectus sapien. Vestibulum viverra nibh at ligula facilisis, ut elementum nibh dignissim. Maecenas suscipit vehicula nibh, eget fringilla erat rutrum vel.",
			ProfileImageURL: fmt.Sprintf("%s/img/fung.jpg", c.staticResourceURL),
			Specialties:     []string{"Psoriasis", "Eczema", "Acne"},
			Experience:      "25 Years",
			ProfileURL:      fmt.Sprintf("%s/md-jason-fung", c.webURL),
		},
		{
			IsSpruceDoctor:  true,
			LongDisplayName: "Dr. Andrew Styperek",
			Description:     "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis vel ipsum quis diam tincidunt lobortis. Pellentesque egestas lectus sapien. Vestibulum viverra nibh at ligula facilisis, ut elementum nibh dignissim. Maecenas suscipit vehicula nibh, eget fringilla erat rutrum vel.",
			ProfileImageURL: fmt.Sprintf("%s/img/andrew.jpg", c.staticResourceURL),
			Specialties:     []string{"Psoriasis", "Eczema", "Acne", "Anti-Aging", "Athlete's Foot or Ringworm", "Bed Bugs", "Canker Sores", "Cold Sores", "Dandruff", "Dry or Itchy Skin", "Eczema", "Excessive Sweating", "Eyelash Thinning", "Hives", "Ingrown Hair", "Lice or Scabies", "Male Hair Loss", "Tick bites"},
			Experience:      "5 Years",
			ProfileURL:      fmt.Sprintf("%s/md-jason-fung", c.webURL),
		},
		{
			IsSpruceDoctor:  false,
			LongDisplayName: "Dr. Lavanya Krishnan",
			Description:     "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis vel ipsum quis diam tincidunt lobortis. Pellentesque egestas lectus sapien. Vestibulum viverra nibh at ligula facilisis, ut elementum nibh dignissim. Maecenas suscipit vehicula nibh, eget fringilla erat rutrum vel.Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis vel ipsum quis diam tincidunt lobortis. Pellentesque egestas lectus sapien. Vestibulum viverra nibh at ligula facilisis, ut elementum nibh dignissim. Maecenas suscipit vehicula nibh, eget fringilla erat rutrum vel.Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis vel ipsum quis diam tincidunt lobortis. Pellentesque egestas lectus sapien. Vestibulum viverra nibh at ligula facilisis, ut elementum nibh dignissim. Maecenas suscipit vehicula nibh, eget fringilla erat rutrum vel.",
			ProfileImageURL: fmt.Sprintf("%s/img/lavanya.jpg", c.staticResourceURL),
			Experience:      "10 Years",
			ProfileURL:      fmt.Sprintf("%s/md-lavanya-krishnan", c.webURL),
		},
		{
			IsSpruceDoctor:  false,
			LongDisplayName: "Dr. George Bluth Senior Junior Senior Junior Senior Junior",
			Description:     "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis vel ipsum quis diam tincidunt lobortis. Pellentesque egestas lectus sapien. Vestibulum viverra nibh at ligula facilisis, ut elementum nibh dignissim. Maecenas suscipit vehicula nibh, eget fringilla erat rutrum vel.",
			ProfileImageURL: fmt.Sprintf("%s/img/beth.jpg", c.staticResourceURL),
			Experience:      "10 Years",
			ProfileURL:      fmt.Sprintf("%s/md-lavanya-krishnan", c.webURL),
		},
	}

	// city := mux.Vars(ctx)["city"]
	www.TemplateResponse(w, http.StatusOK, c.refTemplate, &www.BaseTemplateContext{
		Environment: environment.GetCurrent(),
		Title:       template.HTML("San Francisco"),
		SubContext: map[string]interface{}{
			"Title":          "Top Dermatologists in San Francisco",
			"Description":    "We’ve picked the top dermatologists accepting new patients within 15 miles of San Francisco based on patient ratings, years of experience, and education background.",
			"BannerImageURL": "http://test-spruce-storage.s3.amazonaws.com/sf.jpg",
			"Doctors":        doctors,
			"UVRatingSection": spruceScoreSection{
				Score:       "5.8",
				Description: "Compared to other U.S. cities San Francisco has:",
				Bullets: []string{
					"Higher than U.S. average of 4.9",
					"Apply sunscreen with SPF 30+ daily",
					"Avoid sun exposure  10am-4pm",
				},
			},
			"CareRatingSection": spruceScoreSection{
				Score:       "B+",
				Description: "Compared to other U.S. cities San Francisco has:",
				Bullets: []string{
					"Above average density of dermatologists ",
					"Longer than average wait times",
					"Options for remote or online treatment",
				},
			},
			"TopSkinConditionsSection": descriptionItemsSection{
				Description: "Top 5 skin conditions people seek treatment for in San Francisco are:",
				Items: []string{
					"Acne",
					"Eyelash Thinning",
					"Moles, Spots and Other Growths",
					"Rash",
					"Bed Bugs",
				},
			},
			"NearbyCitiesSection": descriptionItemsSection{
				Items: []string{
					"Oakland, CA",
					"Daly City, CA",
					"San Mateo, CA",
					"Berkeley, CA",
					"Palo Alto, CA",
					"Hayward, CA",
				},
			},
		},
	})
}
