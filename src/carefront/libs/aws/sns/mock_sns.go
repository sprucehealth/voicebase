package sns

type MockSNS struct {
	PushEndpointToReturn string
}

func (m *MockSNS) CreatePlatformEndpoint(platformEndpointArn, token string) (string, error) {
	return m.PushEndpointToReturn, nil
}

func (m *MockSNS) DeleteEndpoint(endpointArn string) error {
	return nil
}

func (m *MockSNS) Publish(message, targetArn string) error {
	return nil
}

func (m *MockSNS) SubscribePlatformEndpointToTopic(platformEndpointArn, topicArn string) error {
	return nil
}
