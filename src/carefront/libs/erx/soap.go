package erx

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"net/http"
)

const (
	xmlContentType = "text/xml; charset=utf-8"
)

type soapAPIData interface {
	GetSoapAction() string
	GetSoapAPIEndPoint() string
}

type soapEnvelope struct {
	XMLName  xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	SOAPBody soapBody `xml:"Body"`
}

type soapBody struct {
	RequestBody []byte `xml:",innerxml"`
}

func makeSoapRequest(requestMessage soapAPIData, result interface{}) error {
	envelope := soapEnvelope{}
	envelope.SOAPBody = soapBody{}
	requestBody, err := xml.Marshal(requestMessage)
	if err != nil {
		return err
	}
	envelope.SOAPBody.RequestBody = requestBody

	envelopBytes, err := xml.Marshal(&envelope)
	if err != nil {
		return err
	}

	buffer := new(bytes.Buffer)
	buffer.WriteString(xml.Header)
	buffer.Write(envelopBytes)

	req, err := http.NewRequest("POST", requestMessage.GetSoapAPIEndPoint(), buffer)
	req.Header.Set("Content-Type", xmlContentType)
	req.Header.Set("SOAPAction", requestMessage.GetSoapAction())

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	responseEnvelope := &soapEnvelope{}
	err = xml.Unmarshal(body, responseEnvelope)
	if err != nil {
		return err
	}

	err = xml.Unmarshal(responseEnvelope.SOAPBody.RequestBody, result)
	if err != nil {
		return err
	}

	return err
}
