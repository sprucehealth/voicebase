package erx

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
)

const (
	xmlContentType = "text/xml; charset=utf-8"
)

type soapEnvelope struct {
	XMLName  xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	SOAPBody soapBody `xml:"Body"`
}

type soapBody struct {
	RequestBody []byte `xml:",innerxml"`
}

type soapClient struct {
	SoapAPIEndPoint string
	APIEndpoint     string
}

func (s *soapClient) makeSoapRequest(soapAction string, requestMessage interface{}, result interface{}, statLatency metrics.Histogram, statRequest, statFailure *metrics.Counter) error {
	envelope := soapEnvelope{
		SOAPBody: soapBody{},
	}
	requestBody, err := xml.Marshal(requestMessage)
	if err != nil {
		return err
	}
	envelope.SOAPBody.RequestBody = requestBody

	buffer := new(bytes.Buffer)
	buffer.WriteString(xml.Header)
	if err := xml.NewEncoder(buffer).Encode(&envelope); err != nil {
		return err
	}

	startTime := time.Now()
	req, err := http.NewRequest("POST", s.SoapAPIEndPoint, buffer)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", xmlContentType)
	req.Header.Set("SOAPAction", s.APIEndpoint+soapAction)

	statRequest.Inc(1)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		statFailure.Inc(1)
		return err
	}
	defer resp.Body.Close()

	responseTime := time.Since(startTime).Nanoseconds() / 1e3
	statLatency.Update(responseTime)

	responseEnvelope := &soapEnvelope{}
	if err := xml.NewDecoder(resp.Body).Decode(responseEnvelope); err != nil {
		statFailure.Inc(1)
		return err
	}

	if err := xml.Unmarshal(responseEnvelope.SOAPBody.RequestBody, result); err != nil {
		statFailure.Inc(1)
		return err
	}

	return nil
}
