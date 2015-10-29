package handlers

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/dal"

	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type urlItem struct {
	Loc        string `xml:"loc"`
	ChangeFreq string `xml:"changefreq"`
}

type urlSet struct {
	URLs []*urlItem `xml:"url"`
}

type sitemap struct {
	URLSet urlSet `xml:"http://www.sitemaps.org/schemas/sitemap/0.9 urlset"`
}

// cachedData represents a cached version of the sitemap
// with an indication of when it was last refreshed
type cachedContent struct {
	lastRefreshed string
	data          []byte
}

type siteMapHandler struct {
	webURL        string
	doctorDAL     dal.DoctorDAL
	cityDAL       dal.CityDAL
	cachedSitemap atomic.Value
}

func NewSiteMapHandler(webURL string, doctorDAL dal.DoctorDAL, cityDAL dal.CityDAL) httputil.ContextHandler {
	return &siteMapHandler{
		doctorDAL: doctorDAL,
		cityDAL:   cityDAL,
		webURL:    webURL,
	}
}

func (s *siteMapHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	if httputil.CheckAndSetETag(w, r, httputil.GenETag(time.Now().Format("2006-01-02")+":cfsitemap")) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	data, err := s.retrieveSiteMap()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.CacheHeaders(w.Header(), time.Time{}, 24*time.Hour)
	w.Header().Set("Content-Type", "application/xml")
	if _, err := w.Write(data); err != nil {
		golog.Errorf(err.Error())
	}
}

func (s *siteMapHandler) retrieveSiteMap() ([]byte, error) {
	currentDateStr := time.Now().Format("2006-01-02")
	// return cached version of data if date since last refresh
	// has not changed
	c := s.cachedSitemap.Load()
	if c != nil && c.(*cachedContent).lastRefreshed == currentDateStr {
		return c.(*cachedContent).data, nil
	}

	var cityIDs, doctorIDs []string
	p := conc.NewParallel()

	p.Go(func() error {
		var err error
		doctorIDs, err = s.doctorDAL.ShortListedDoctorIDs()
		if err != nil {
			golog.Errorf("Unable to get short list of doctor ids: %s", err.Error())
			return errors.Trace(err)
		}

		return nil
	})

	p.Go(func() error {
		var err error
		cityIDs, err = s.cityDAL.ShortListedCityIDs()
		if err != nil {
			golog.Errorf("Unable to get short list of city ids: %s", err.Error())
			return errors.Trace(err)
		}

		return nil
	})

	if err := p.Wait(); err != nil {
		return nil, errors.Trace(err)
	}

	sm := &sitemap{
		URLSet: urlSet{
			URLs: make([]*urlItem, 0, len(cityIDs)+len(doctorIDs)),
		},
	}

	// indicating daily change for doctor and city pages
	// as the yelp reviews and uv index have the potential to change daily
	for _, item := range doctorIDs {
		sm.URLSet.URLs = append(sm.URLSet.URLs, &urlItem{
			Loc:        fmt.Sprintf("%s/%s", s.webURL, item),
			ChangeFreq: "daily",
		})
	}
	for _, item := range cityIDs {
		sm.URLSet.URLs = append(sm.URLSet.URLs, &urlItem{
			Loc:        fmt.Sprintf("%s/%s", s.webURL, item),
			ChangeFreq: "daily",
		})
	}

	xmlData, err := xml.Marshal(sm)
	if err != nil {
		return nil, errors.Trace(err)
	}

	xmlOutput := fmt.Sprintf("%s%s", xml.Header, string(xmlData))
	s.cachedSitemap.Store(&cachedContent{
		data:          []byte(xmlOutput),
		lastRefreshed: currentDateStr,
	})
	return []byte(xmlOutput), nil
}
