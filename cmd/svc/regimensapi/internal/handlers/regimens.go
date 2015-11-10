package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/media"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/rxguide"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/imageutil"
	"github.com/sprucehealth/backend/libs/imageutil/collage"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/svc/regimens"
	"github.com/sprucehealth/schema"
	"golang.org/x/net/context"
)

const (
	productPlaceholderMediaID = "product_placeholder.png"
)

type regimensHandler struct {
	svc                regimens.Service
	deterministicStore storage.DeterministicStore
	webDomain          string
	apiDomain          string
}

// NewRegimens returns a new regimens search and manipulation handler.
func NewRegimens(svc regimens.Service, deterministicStore storage.DeterministicStore, webDomain, apiDomain string) httputil.ContextHandler {
	return httputil.SupportedMethods(&regimensHandler{
		svc:                svc,
		deterministicStore: deterministicStore,
		webDomain:          webDomain,
		apiDomain:          apiDomain,
	}, httputil.Get, httputil.Post)
}

func (h *regimensHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		rd, err := h.parseGETRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		h.serveGET(ctx, w, r, rd)
	case httputil.Post:
		rd, err := h.parsePOSTRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		h.servePOST(ctx, w, r, rd)
	}
}

func (h *regimensHandler) parseGETRequest(ctx context.Context, r *http.Request) (*responses.RegimensGETRequest, error) {
	rd := &responses.RegimensGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *regimensHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *responses.RegimensGETRequest) {
	tags := strings.Fields(rd.Query)
	for i, t := range tags {
		tags[i] = strings.ToLower(strings.TrimLeft(t, "#"))
	}

	// If there are no tags return an empty result
	if len(tags) == 0 {
		httputil.JSONResponse(w, http.StatusOK, &responses.RegimensGETResponse{})
		return
	}

	// Arbitrarily limit this till we understand the implications of tag filtering
	if len(tags) > 5 {
		apiservice.WriteBadRequestError(ctx, fmt.Errorf("A maximum number of 5 tags can be used in a single query. %d provided", len(tags)), w, r)
		return
	}

	regimens, err := h.svc.TagQuery(tags)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	fillMissingProductMedia(h.apiDomain, regimens)

	// Cache regimens tag queries for 5 minutes
	httputil.CacheHeaders(w.Header(), time.Time{}, 5*time.Minute)
	httputil.JSONResponse(w, http.StatusOK, &responses.RegimensGETResponse{Regimens: regimens})
}

func (h *regimensHandler) parsePOSTRequest(ctx context.Context, r *http.Request) (*responses.RegimenPOSTRequest, error) {
	rd := &responses.RegimenPOSTRequest{}
	// An empty body for a POST here is acceptable
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil && err != io.EOF {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *regimensHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *responses.RegimenPOSTRequest) {
	if err := validateRegimenContents(rd.Regimen); err != nil && !rd.AllowRestricted {
		apiservice.WriteBadRequestError(ctx, err, w, r)
		return
	}

	var resourceID, authToken string
	var regimen *regimens.Regimen
	if rd.Regimen == nil || rd.Regimen.ID == "" {
		var err error
		resourceID, err = newShortID()
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		authToken, err = h.svc.AuthorizeResource(resourceID)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		// Write an empty regimen to the store to bootstrap it if one wasn't provided
		url := regimenURL(h.webDomain, resourceID)
		if rd.Regimen == nil {
			regimen = &regimens.Regimen{ID: resourceID, URL: url, CoverPhotoURL: media.ResizeURL(h.apiDomain, productPlaceholderMediaID, collageWidth, collageFallbackHeight)}
		} else {
			regimen = rd.Regimen
			regimen.ID = resourceID
			regimen.URL = url
		}
	} else if rd.Regimen.ID != "" {
		// If they provided a regimen ID, make sure they can access it and it isn't published
		resourceID = rd.Regimen.ID
		authToken = r.Header.Get("token")
		if authToken == "" {
			apiservice.WriteAccessNotAllowedError(ctx, w, r)
			return
		}
		access, err := h.svc.CanAccessResource(rd.Regimen.ID, authToken)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		} else if !access {
			apiservice.WriteAccessNotAllowedError(ctx, w, r)
			return
		}

		_, published, err := h.svc.Regimen(resourceID)
		if api.IsErrNotFound(err) {
			apiservice.WriteResourceNotFoundError(ctx, err.Error(), w, r)
			return
		} else if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		} else if published {
			apiservice.WriteAccessNotAllowedError(ctx, w, r)
			return
		}
	}

	if regimen == nil || regimen.ID == "" {
		apiservice.WriteError(ctx, errors.New("The regimen preparing to be written is null or lacks an identifier"), w, r)
		return
	}

	// We can't associate a regimen with more than 24 tags
	if len(regimen.Tags) > 24 {
		apiservice.WriteBadRequestError(ctx, errors.New("A regimen can only be associated with a maximum of 24 tags"), w, r)
		return
	}

	// Normalize the tags
	for i, t := range regimen.Tags {
		rd.Regimen.Tags[i] = strings.ToLower(t)
	}

	// Generate a collage if we don't have a cover image, it is a previous collage, or it is the placeholder image
	if regimen.CoverPhotoURL == "" || strings.HasSuffix(regimen.CoverPhotoURL, collageSuffix) {
		collageURL, width, height, err := generateCollage(resourceID, rd.Regimen, h.deterministicStore, h.apiDomain)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
		rd.Regimen.CoverPhotoURL = collageURL
		rd.Regimen.CoverPhoto = &regimens.Image{URL: collageURL, Width: width, Height: height}
	} else {
		width, height := remoteImageDimensions(regimen.CoverPhotoURL)
		regimen.CoverPhoto = &regimens.Image{URL: regimen.CoverPhotoURL, Width: width, Height: height}
	}

	if err := h.svc.PutRegimen(regimen.ID, regimen, rd.Publish); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &responses.RegimenPOSTResponse{
		ID:        resourceID,
		URL:       regimenURL(h.webDomain, resourceID),
		AuthToken: authToken,
	})
}

type regimenHandler struct {
	svc                regimens.Service
	deterministicStore storage.DeterministicStore
	webDomain          string
	apiDomain          string
}

// NewRegimen returns a new regimen search and manipulation handler.
func NewRegimen(svc regimens.Service, deterministicStore storage.DeterministicStore, webDomain, apiDomain string) httputil.ContextHandler {
	return httputil.SupportedMethods(&regimenHandler{
		svc:                svc,
		deterministicStore: deterministicStore,
		webDomain:          webDomain,
		apiDomain:          apiDomain,
	}, httputil.Get, httputil.Put)
}

func (h *regimenHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id, ok := mux.Vars(ctx)["id"]
	if !ok {
		apiservice.WriteResourceNotFoundError(ctx, "an id must be provided", w, r)
		return
	}
	regimen, published, err := h.svc.Regimen(id)
	if api.IsErrNotFound(err) {
		apiservice.WriteResourceNotFoundError(ctx, err.Error(), w, r)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// If this is a mutating request or a GET on an unpublished record check auth
	// If there is no token in the header check the params
	authToken := r.Header.Get("token")
	if authToken == "" && r.Method == httputil.Get {
		rd, err := h.parseGETRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		authToken = rd.AuthToken
	}
	if r.Method == httputil.Put || (r.Method == httputil.Get && !published) {
		access, err := h.svc.CanAccessResource(id, authToken)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		} else if !access {
			apiservice.WriteAccessNotAllowedError(ctx, w, r)
			return
		}
	}

	switch r.Method {
	case httputil.Get:
		h.serveGET(ctx, w, r, regimen, published)
	case httputil.Put:
		// Do not allow published regimens to be mutated
		if published {
			apiservice.WriteAccessNotAllowedError(ctx, w, r)
			return
		}
		rd, err := h.parsePUTRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		h.servePUT(ctx, w, r, rd, id)
	}
}

func (h *regimenHandler) parseGETRequest(ctx context.Context, r *http.Request) (*responses.RegimenGETRequest, error) {
	rd := &responses.RegimenGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *regimenHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request, regimen *regimens.Regimen, published bool) {
	if published {
		// Cache published regimen queries for 5 minutes
		httputil.CacheHeaders(w.Header(), time.Time{}, 5*time.Minute)
	} else {
		// Never cache an unpublished regimen as it is subject to change
		httputil.NoCache(w.Header())
	}
	fillMissingProductMedia(h.apiDomain, []*regimens.Regimen{regimen})
	httputil.JSONResponse(w, http.StatusOK, regimen)
}

func (h *regimenHandler) parsePUTRequest(ctx context.Context, r *http.Request) (*responses.RegimenPUTRequest, error) {
	rd := &responses.RegimenPUTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.Regimen == nil {
		return nil, fmt.Errorf("regimen required")
	}
	return rd, nil
}

func (h *regimenHandler) servePUT(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *responses.RegimenPUTRequest, resourceID string) {
	if err := validateRegimenContents(rd.Regimen); err != nil && !rd.AllowRestricted {
		apiservice.WriteBadRequestError(ctx, err, w, r)
		return
	}

	authToken := r.Header.Get("token")
	for i, t := range rd.Regimen.Tags {
		rd.Regimen.Tags[i] = strings.ToLower(t)
	}
	rd.Regimen.ID = resourceID
	rd.Regimen.URL = regimenURL(h.webDomain, resourceID)

	// We can't associate a regimen with more than 24 tags
	if len(rd.Regimen.Tags) > 24 {
		apiservice.WriteBadRequestError(ctx, errors.New("A regimen can only be associated with a maximum of 24 tags"), w, r)
		return
	}

	// Generate a collage if we don't have a cover image, it is a previous collage, or it is the placeholder image
	if rd.Regimen.CoverPhotoURL == "" || strings.HasSuffix(rd.Regimen.CoverPhotoURL, collageSuffix) || strings.HasSuffix(rd.Regimen.CoverPhotoURL, productPlaceholderMediaID) {
		collageURL, width, height, err := generateCollage(resourceID, rd.Regimen, h.deterministicStore, h.apiDomain)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
		rd.Regimen.CoverPhotoURL = collageURL
		rd.Regimen.CoverPhoto = &regimens.Image{URL: collageURL, Width: width, Height: height}
	} else {
		width, height := remoteImageDimensions(rd.Regimen.CoverPhotoURL)
		rd.Regimen.CoverPhoto = &regimens.Image{URL: rd.Regimen.CoverPhotoURL, Width: width, Height: height}
	}

	if err := h.svc.PutRegimen(resourceID, rd.Regimen, rd.Publish); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &responses.RegimenPOSTResponse{
		ID:        resourceID,
		URL:       regimenURL(h.webDomain, resourceID),
		AuthToken: authToken,
	})
}

const (
	collageWidth             = 500
	collageSuffix            = "_spruce_product_collage"
	collageImageHeightScalar = 0.8
	collageImageWidthScalar  = 0.8
	collageCenterRowIsolated = true
	collageFallbackHeight    = 500
)

// remoteImageDimensions will make a best effort attempt at determining the dimensions of the provided image.
// if any errors are encountered then a width and height of 0x0 will be returned
func remoteImageDimensions(u string) (int, int) {
	res, err := http.Get(u)
	if err != nil || res.StatusCode != 200 {
		if res != nil {
			golog.Warningf("Error while attempting to fetch url %s, status code: %d, err: %s", u, res.StatusCode, err)
			return 0, 0
		}
		golog.Warningf("Error while attempting to fetch url %s, err: %s", u, err)
		return 0, 0
	}
	defer res.Body.Close()
	c, _, err := image.DecodeConfig(res.Body)
	if err != nil {
		golog.Warningf("Error while decoding image config %s: %s", u, err)
		return 0, 0
	}
	return c.Width, c.Height
}

// TODO: We could optimize this flow by only reading in one image at a time as we add it to the collage, mark for future performance improvement
func generateCollage(resourceID string, r *regimens.Regimen, deterministicStore storage.DeterministicStore, apiDomain string) (string, int, int, error) {
	var images []image.Image
ProductImageLoop:
	for _, ps := range r.ProductSections {
		for _, p := range ps.Products {
			// Arbitrarily limit the number of images in a collage to 9
			if len(images) == 9 {
				break ProductImageLoop
			}
			if p.ImageURL != "" && !strings.HasSuffix(p.ImageURL, rxguide.RXPlaceholderMediaID) {
				res, err := http.Get(p.ImageURL)
				if err != nil || res.StatusCode != 200 {
					if res != nil {
						golog.Warningf("Error while attempting to fetch url %s, status code: %d, err: %s", p.ImageURL, res.StatusCode, err)
						continue
					}
					golog.Warningf("Error while attempting to fetch url %s, err: %s", p.ImageURL, err)
					continue
				}
				defer res.Body.Close()
				m, _, err := image.Decode(res.Body)
				if err != nil {
					golog.Warningf("Error while decoding image %s: %s", p.ImageURL, err)
					continue
				}
				images = append(images, m)
			}
		}
	}
	if len(images) == 0 {
		golog.Warningf("No usable images were found in regimen")
		return media.ResizeURL(apiDomain, productPlaceholderMediaID, collageWidth, collageFallbackHeight), collageWidth, collageFallbackHeight, nil
	}
	result, err := collage.Collageify(images, collage.SpruceProductGridLayout, &collage.Options{
		Width:             collageWidth,
		ImageHeightScalar: collageImageHeightScalar,
		ImageWidthScalar:  collageImageWidthScalar,
		CenterRowIsolated: collageCenterRowIsolated,
	})
	if err != nil {
		golog.Errorf("Unable to generate collage from product images for regimen %s - Falling back to placeholder: %s", resourceID, err)
		return media.ResizeURL(apiDomain, productPlaceholderMediaID, collageWidth, collageFallbackHeight), collageWidth, collageFallbackHeight, nil
	}
	buf := bytes.NewBuffer(nil)
	if err := jpeg.Encode(buf, result, &jpeg.Options{Quality: imageutil.JPEGQuality}); err != nil {
		return "", 0, 0, errors.Trace(err)
	}
	_, err = deterministicStore.Put("m"+resourceID+collageSuffix, buf.Bytes(), "image/jpeg", nil)
	return media.URL(apiDomain, resourceID+collageSuffix), result.Bounds().Dx(), result.Bounds().Dy(), errors.Trace(err)
}

// Apply changes to a list of regimens that populate plateholder data
// Note: This is intended to be used on GET requests after getting the info from
//   the data store. This is to not lock us into these urls in the actual data
func fillMissingProductMedia(apiDomain string, rs []*regimens.Regimen) {
	for _, r := range rs {
		for _, ps := range r.ProductSections {
			for _, p := range ps.Products {
				if p.ImageURL == "" {
					p.ImageURL = media.ResizeURL(apiDomain, productPlaceholderMediaID, 100, 100)
				}
			}
		}
	}
}

func regimenURL(webDomain, resourceID string) string {
	return webDomain + "/regimen/" + resourceID
}

var (
	restrictedTags = map[string]bool{
		"#dermatologistown":         true,
		"dermatologistown":          true,
		"#dermatologistrecommended": true,
		"dermatologistrecommended":  true,
		"#createdbyspruce":          true,
		"createdbyspruce":           true,
	}
)

func validateRegimenContents(r *regimens.Regimen) error {
	if r == nil {
		return nil
	}

	for _, t := range r.Tags {
		if err := validateTag(t); err != nil {
			return err
		}
	}
	if err := validateUsername(r.Creator.Name); err != nil {
		return err
	}

	if r.CoverPhotoURL != "" {
		if err := validateURL(r.CoverPhotoURL); err != nil {
			return errors.Trace(err)
		}
	}

	for _, ps := range r.ProductSections {
		for _, p := range ps.Products {
			if p.ImageURL != "" {
				if err := validateURL(p.ImageURL); err != nil {
					return errors.Trace(err)
				}
				if err := validateURL(p.ProductURL); err != nil {
					return errors.Trace(err)
				}
			}
		}
	}
	return nil
}

func validateURL(u string) error {
	if _, err := url.Parse(u); err != nil {
		return errors.Trace(fmt.Errorf("%s is not a valid URL", u))
	}
	return nil
}

func validateTag(tag string) error {
	if _, ok := restrictedTags[strings.ToLower(tag)]; ok {
		return fmt.Errorf("tag: %s is not allowed for public use", tag)
	}
	return nil
}

func validateUsername(username string) error {
	if strings.Contains(strings.ToLower(username), "spruce") {
		return errors.New("Usernames cannot contain the term 'spruce'")
	}
	return nil
}

func newShortID() (string, error) {
	id, err := idgen.NewID()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(base64.URLEncoding.EncodeToString([]byte{
		byte(id >> 56), byte(id >> 48), byte(id >> 40), byte(id >> 32),
		byte(id >> 24), byte(id >> 16), byte(id >> 8), byte(id),
	}), "="), nil
}
