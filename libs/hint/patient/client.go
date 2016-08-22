package patient

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/sprucehealth/backend/libs/hint"
)

type Client struct {
	B   hint.Backend
	Key string
}

func New(practiceKey string, params *hint.PatientParams) (*hint.Patient, error) {
	return getC().New(practiceKey, params)
}

func Get(practiceKey, id string) (*hint.Patient, error) {
	return getC().Get(practiceKey, id)
}

func Update(practiceKey, id string, params *hint.PatientParams) (*hint.Patient, error) {
	return getC().Update(practiceKey, id, params)
}

func Delete(practiceKey, id string) error {
	return getC().Delete(practiceKey, id)
}

func List(practiceKey string, params *hint.ListParams) *hint.Iter {
	return getC().List(practiceKey, params)
}

func (c Client) New(practiceKey string, params *hint.PatientParams) (*hint.Patient, error) {
	if practiceKey == "" {
		return nil, errors.New("practice_key required")
	}

	patient := &hint.Patient{}
	if _, err := c.B.Call("POST", "/provider/patients", practiceKey, params, patient); err != nil {
		return nil, err
	}

	return patient, nil
}

func (c Client) Get(practiceKey, id string) (*hint.Patient, error) {
	if practiceKey == "" {
		return nil, errors.New("practice_key required")
	}

	patient := &hint.Patient{}
	if _, err := c.B.Call("GET", fmt.Sprintf("/provider/patients/%s", id), practiceKey, nil, patient); err != nil {
		return nil, err
	}
	return patient, nil
}

func (c Client) Update(practiceKey, id string, params *hint.PatientParams) (*hint.Patient, error) {
	if practiceKey == "" {
		return nil, errors.New("practice_key required")
	}

	patient := &hint.Patient{}
	if _, err := c.B.Call("PATCH", fmt.Sprintf("/provider/patients/%s", id), practiceKey, params, patient); err != nil {
		return nil, err
	}

	return patient, nil
}

func (c Client) Delete(practiceKey, id string) error {
	if practiceKey == "" {
		return errors.New("practice_key required")
	}

	if _, err := c.B.Call("DELETE", fmt.Sprintf("/provider/patients/%s", id), practiceKey, nil, nil); err != nil {
		return err
	}

	return nil
}

func (c Client) List(practiceKey string, params *hint.ListParams) *hint.Iter {
	return hint.GetIter(params, func(lp *hint.ListParams) ([]interface{}, hint.ListMeta, error) {
		var meta hint.ListMeta

		encodedParams, err := lp.Encode()
		if err != nil {
			return nil, meta, err
		}

		var patients []*hint.Patient
		resHeaders, err := c.B.Call("GET", fmt.Sprintf("provider/patients?%s", encodedParams), practiceKey, nil, &patients)
		if err != nil {
			return nil, meta, err
		}

		if xCountHeader := resHeaders.Get("x-count"); xCountHeader != "" {
			meta.CurrentCount, err = strconv.ParseUint(xCountHeader, 10, 64)
			if err != nil {
				return nil, meta, err
			}
		}

		if xTotalCountHeader := resHeaders.Get("x-total-count"); xTotalCountHeader != "" {
			meta.TotalCount, err = strconv.ParseUint(xTotalCountHeader, 10, 64)
			if err != nil {
				return nil, meta, err
			}
		}

		ret := make([]interface{}, len(patients))
		for i, patient := range patients {
			ret[i] = patient
		}

		return ret, meta, nil
	})
}

func getC() Client {
	return Client{hint.GetBackend(), hint.Key}
}
