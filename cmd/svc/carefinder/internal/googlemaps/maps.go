package googlemaps

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/libs/sig"
)

type ImageFormat string

const (
	ImageFormatPNG8        ImageFormat = "png"
	ImageFormatPNG32       ImageFormat = "png32"
	ImageFormatGIF         ImageFormat = "gif"
	ImageFormatJPG         ImageFormat = "jpg"
	ImageFormatJPGBaseline ImageFormat = "jpg-baseline"
)

type mapType string

const (
	MapTypeRoadmap   mapType = "roadmap"
	MapTypeSatellite mapType = "satellite"
	MapTypeTerrain   mapType = "terrain"
	MapTypeHybrid    mapType = "hybrid"
)

// StaticMapConfig defines a list of possible parameters
// that can be provided to the google static maps API
// to generate an image.
// Learn more about it here:
// https://developers.google.com/maps/documentation/static-maps/intro#Markers
type StaticMapConfig struct {

	// Center defines the center of the map image. It is optional
	// if atlease one marker is specified and required if not.
	Center Coordinates

	// Zoom specifies the zoom level of the map. An acceptable value is
	// between 0 and 21.
	Zoom int

	// Width defines the width of the image requested (in px).
	Width int

	// Height defines the height of the image requested (in px).
	Height int

	// Scale is an optional parameter that represents the number
	// of pixels that are returned. If scale 2 then twice as many
	// pixels are returned which works well for high resolution
	// screens.
	Scale int

	// Format defines the format of the resulting image. By,
	// default format is PNG.
	Format ImageFormat

	// Markers allow definition of one or markers to be defined
	// at specified locations on the map. Center and zoom are
	// not required if a marker is specified. Google will
	// implicitly determine the zoom and center to get the markers
	// to fit on the map.
	Markers []MarkerConfig

	// MapType defines the type of map to construct.
	MapType mapType

	// API key for google static map api.
	Key string

	URLSigningKey string
}

func (s *StaticMapConfig) Encode() string {
	u := url.Values{}
	if !s.Center.IsZero() {
		u.Set("center", s.Center.String())
	}
	if s.Zoom > 0 {
		u.Set("zoom", strconv.Itoa(s.Zoom))
	}
	if s.Width > 0 && s.Height > 0 {
		u.Set("size", fmt.Sprintf("%dx%d", s.Width, s.Height))
	}
	if s.Scale > 0 {
		u.Set("scale", strconv.Itoa(s.Scale))
	}
	if string(s.Format) != "" {
		u.Set("format", string(s.Format))
	}
	for _, m := range s.Markers {
		u.Set("markers", m.Encode())
	}
	u.Set("key", s.Key)
	return u.Encode()
}

type MarkerSizeType string

const (
	MarkerSizeTypeTiny   MarkerSizeType = "tiny"
	MarkerSizeTypeMid    MarkerSizeType = "mid"
	MarkerSizeTypeNormal MarkerSizeType = "normal"
)

const (
	ColorBlack  = "black"
	ColorBrown  = "brown"
	ColorGreem  = "green"
	ColorPurple = "purple"
	ColorYellow = "yellow"
	ColorBlue   = "blue"
	ColorGray   = "gray"
	ColorOrange = "orange"
	ColorRed    = "red"
	ColorWhite  = "white"
)

type MarkerConfig struct {
	Size      MarkerSizeType
	Color     string
	Locations []Coordinates
}

func (m *MarkerConfig) Encode() string {
	parts := make([]string, 0, 2+len(m.Locations))

	if string(m.Size) != "" {
		parts = append(parts, fmt.Sprintf("size:%s", m.Size))
	}
	if m.Color != "" {
		parts = append(parts, fmt.Sprintf("color:%s", m.Color))
	}
	for _, l := range m.Locations {
		parts = append(parts, l.String())
	}

	return strings.Join(parts, "|")
}

// Coordinates defines a point on a map
// identified by its latitude and longitude.
type Coordinates struct {
	Latitude  float64
	Longitude float64
}

func (c *Coordinates) IsZero() bool {
	return c.Latitude == 0.0 && c.Longitude == 0.0
}

func (c *Coordinates) String() string {
	return fmt.Sprintf("%f,%f", c.Latitude, c.Longitude)
}

const (
	api = "https://maps.googleapis.com/maps/api/staticmap"
)

func GenerateImageURL(cfg *StaticMapConfig) (string, error) {

	url := api + "?" + cfg.Encode()
	signature, err := generateSignature(url, cfg.URLSigningKey)

	return url + "&signature=" + signature, err
}

func generateSignature(imageURL, urlSigningSecret string) (string, error) {
	u, err := url.Parse(imageURL)
	if err != nil {
		return "", err
	}

	key, err := base64.URLEncoding.DecodeString(urlSigningSecret)
	if err != nil {
		return "", err
	}

	s, err := sig.NewSigner([][]byte{key}, nil)
	if err != nil {
		return "", err
	}

	signature, err := s.Sign([]byte(u.RequestURI()))
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(signature), nil

}
