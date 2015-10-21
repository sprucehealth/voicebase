package uv

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Service interface {
	DailyUVIndexByCityState(city, state string) (int, error)
}

type service struct{}

func NewService() Service {
	return &service{}
}

type uvData struct {
	City    string `json:"CITY"`
	State   string `json:"STATE"`
	UVIndex int    `json:"UV_INDEX"`
	UVAlert int    `json:"UV_ALERT"`
}

func (s *service) DailyUVIndexByCityState(city, state string) (int, error) {
	resp, err := http.Get(fmt.Sprintf("http://iaspub.epa.gov/enviro/efservice/getEnvirofactsUVDAILY/CITY/%s/STATE/%s/json", city, state))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Unable to read UV Index. Status code: %d", resp.StatusCode)
	}

	var ud []uvData
	if err := json.NewDecoder(resp.Body).Decode(&ud); err != nil {
		return 0.0, err
	} else if err == io.EOF {
		return 0.0, nil
	}

	if len(ud) == 0 {
		return 0.0, nil
	}

	return ud[0].UVIndex, nil
}
